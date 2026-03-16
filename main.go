package main

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"time"

	"github.com/google/uuid"
)

const (
	IFLOW_BASE_URL = "https://apis.iflow.cn/v1" // ✅ убраны пробелы
	PROXY_PORT     = "8318"
	LOG_FILE       = "proxy.log"
)

var (
	apikey      string
	sessionID   string
	logFilePath string
)

type IFlowSettings struct {
	ApiKey string `json:"apiKey"`
}

func getIFlowAPIKey() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("user: %w", err)
	}
	configPath := filepath.Join(usr.HomeDir, ".iflow", "settings.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("read config: %w", err)
	}
	var settings IFlowSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return "", fmt.Errorf("parse config: %w", err)
	}
	if settings.ApiKey == "" {
		return "", fmt.Errorf("API key empty")
	}
	return settings.ApiKey, nil
}

func createSignature(userAgent, sessionID string, timestamp int64, key string) string {
	payload := fmt.Sprintf("%s:%s:%d", userAgent, sessionID, timestamp)
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(payload))
	return hex.EncodeToString(h.Sum(nil))
}

func logToFile(format string, args ...interface{}) {
	f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("log open: %v", err)
		return
	}
	defer f.Close()
	ts := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(f, "[%s] %s\n", ts, fmt.Sprintf(format, args...))
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

// ─── /v1/models ──────────────────────────────────────────────────────────────
type ModelsResponse struct {
	Data   []ModelItem `json:"data"`
	Object string      `json:"object"`
}
type ModelItem struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	OwnedBy string `json:"owned_by"`
	Created int64  `json:"created"`
}

func modelsHandler(w http.ResponseWriter, r *http.Request) {
	logToFile("→ Models request: %s %s (Path: %s)", r.Method, r.URL.Path, r.URL.Path)
	
	if r.URL.Path != "/v1/models" {
		logToFile("✗ Models: Path mismatch - expected /v1/models, got %s", r.URL.Path)
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	
	response := ModelsResponse{
		Object: "list",
		Data: []ModelItem{
			{ID: "glm-5", Object: "model", Created: 1770000000, OwnedBy: "iflow"},
			{ID: "glm-4.7", Object: "model", Created: 1760000000, OwnedBy: "iflow"},
			{ID: "qwen3-coder-plus", Object: "model", Created: 1753228800, OwnedBy: "iflow"},
			{ID: "deepseek-v3.2", Object: "model", Created: 1759104000, OwnedBy: "iflow"},
			{ID: "kimi-k2.5", Object: "model", Created: 1769472000, OwnedBy: "moonshot"},
			{ID: "kimi-k2-thinking", Object: "model", Created: 1762387200, OwnedBy: "moonshot"},
			{ID: "minimax-m2.5", Object: "model", Created: 1750000000, OwnedBy: "minimax"},
		},
	}
	
	// Логируем JSON ответ для отладки
	jsonBytes, _ := json.MarshalIndent(response, "", "  ")
	logToFile("← Models response: %s", string(jsonBytes))
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	logToFile("← Models: 200 OK")
}

// ─── /v1/chat/completions — ПРОСТОЙ ПРОКСИ БЕЗ ТРАНСФОРМАЦИЙ ───────────────
func proxyHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/v1/chat/completions" {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	// Читаем тело запроса "как есть"
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Read body: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	logToFile("→ Request: %s %s", r.Method, r.URL.Path)

	// Готовим запрос к iFlow: тело передаём без изменений
	userAgent := "iFlow-Cli"
	timestamp := time.Now().UnixMilli()
	signature := createSignature(userAgent, sessionID, timestamp, apikey)

	upstreamReq, err := http.NewRequest("POST", IFLOW_BASE_URL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		http.Error(w, "Create upstream: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Заголовки аутентификации для iFlow
	upstreamReq.Header.Set("Content-Type", "application/json")
	upstreamReq.Header.Set("Authorization", "Bearer "+apikey)
	upstreamReq.Header.Set("User-Agent", userAgent)
	upstreamReq.Header.Set("session-id", sessionID)
	upstreamReq.Header.Set("conversation-id", "")
	upstreamReq.Header.Set("x-iflow-timestamp", strconv.FormatInt(timestamp, 10))
	upstreamReq.Header.Set("x-iflow-signature", signature)

	// Выполняем запрос
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(upstreamReq)
	if err != nil {
		http.Error(w, "Upstream: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Пробрасываем заголовки ответа (кроме конфликтующих)
	for k, vals := range resp.Header {
		if k == "Content-Type" || k == "Content-Length" || k == "Transfer-Encoding" {
			continue
		}
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}

	// Определяем, стриминг ли это
	isStream := resp.Header.Get("Content-Type") == "text/event-stream" ||
		bytes.Contains(body, []byte(`"stream":true`))

	if isStream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		// 🔥 ПРОСТО пробрасываем байты, БЕЗ парсинга и модификаций
		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				break
			}
			w.Write(line)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		// 🔥 ПРОСТО копируем тело ответа, БЕЗ парсинга
		io.Copy(w, resp.Body)
	}

	logToFile("← Response: %d", resp.StatusCode)
}

// ─── Main ───────────────────────────────────────────────────────────────────
func main() {
	execPath, _ := os.Executable()
	logFilePath = filepath.Join(filepath.Dir(execPath), LOG_FILE)

	var err error
	apikey, err = getIFlowAPIKey()
	if err != nil {
		log.Fatalf("API key: %v", err)
	}

	sessionID = "session-" + uuid.New().String()

	log.Printf("✓ Simple proxy started (no content transformation)")
	log.Printf("✓ Logging to: %s", logFilePath)

	http.HandleFunc("/v1/chat/completions", corsMiddleware(proxyHandler))
	http.HandleFunc("/v1/models", corsMiddleware(modelsHandler))

	addr := ":" + PROXY_PORT
	fmt.Printf("🚀 iFlow Proxy (SIMPLE) → http://localhost%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}