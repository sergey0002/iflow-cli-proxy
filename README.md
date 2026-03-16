# iFlow Proxy Server

A simple proxy server written in Go that provides unlimited access to GLM-5 and other models available in [iFlow CLI](https://iflow.cn) for your own purposes.

> ⚠️ **WARNING: Use at your own risk!**
> The author is not responsible for the use of this software. You use it entirely at your own risk.

The proxy server uses authorization and endpoints from **iFlow CLI** to provide unlimited API requests to GLM-5 and other models in a format compatible with OpenAI API.

## ⚠️ Important Limitations

- **Works only on Windows** - currently the proxy server is configured to work only on Windows operating system
- On Mac and Linux, the paths to CLI key files may differ. If desired, you can investigate and extend the source code to support these operating systems

## 🔧 Editing Source Code

If you want to change settings (ports, paths, etc.):

1. Make sure **Go** is installed on your computer
2. Edit the [`main.go`](main.go) file as needed
3. Run the [`rebuild-and-start.bat`](rebuild-and-start.bat) file
   - It will automatically find and stop the running process
   - Recompile the program
   - Start the proxy server with new settings

## Features

- ✅ OpenAI-compatible API (`/v1/chat/completions` format)
- ✅ Unlimited requests to models including GLM5 via iFlow CLI
- ✅ Streaming support
- ✅ Automatic authorization via iFlow CLI settings (installed on your PC)
- ✅ CORS support for web applications

## Supported Models

- `glm-5` - main model
- `glm-4.7`
- `qwen3-coder-plus`
- `deepseek-v3.2`
- `kimi-k2.5`
- `kimi-k2-thinking`
- `minimax-m2.5`

## Installation and Setup for Kilo Code

### Step 1: Install iFlow CLI

Download and install iFlow CLI from the official website: https://iflow.cn

### Step 2: Authorize in iFlow CLI

Open a terminal and run the authorization command:

```bash
iflow login
```

Follow the instructions to log in to your iFlow account.

### Step 3: Start the Proxy Server

Two options are available to start the proxy server:

**Option 1: Quick Start (without recompilation)**
```bash
start.bat
```
This file kills the old process and starts the already compiled `iflow-proxy.exe`.

**Option 2: Recompile and Start**
```bash
rebuild-and-start.bat
```
This file recompiles the program and starts it.

The proxy server will start at: **http://127.0.0.1:8318**

### Step 4: Configure Kilo Code

![Kilo Code Setup](img.jpg)

1. Open Kilo Code settings
2. In the **API Endpoint** field, specify:
   ```
   http://127.0.0.1:8318/v1
   ```
3. In the **API Token** field, enter **any value** (e.g., `dummy-token`)
   - The token is not verified by the proxy server, authorization occurs via iFlow CLI
4. Select a model: `glm-5` or any other from the supported list

## API Endpoints

### Get List of Models

```bash
GET http://127.0.0.1:8318/v1/models
```

**Example curl request:**
```bash
curl http://localhost:8318/v1/models
```

### Chat Completions (OpenAI-compatible)

```bash
POST http://127.0.0.1:8318/v1/chat/completions
```

**Example curl request:**
```bash
curl http://localhost:8318/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "glm-4.7",
    "messages": [{"role": "user", "content": "Hello!"}],
    "stream": false
  }'
```

**Example JSON request:**
```json
{
  "model": "glm-5",
  "messages": [
    {
      "role": "user",
      "content": "Hello! How are you?"
    }
  ],
  "stream": true
}
```

## How It Works

1. The proxy server automatically reads the API key from `~/.iflow/settings.json`
2. Upon receiving a request, it forms an HMAC-SHA256 signature for authorization in iFlow
3. Forwards the request to iFlow API without modifying the content
4. Returns the response in a format compatible with OpenAI API

## Logging

All requests and responses are logged to the `proxy.log` file in the proxy server startup directory.

## Requirements

- **Windows** operating system
- Installed iFlow CLI with active authorization
- Go 1.21+ (only for editing and recompiling source code)

## Ports

- **8318** - default proxy server port

## Troubleshooting

### Error "API key: read config: no such file or directory"

Make sure iFlow CLI is installed and you are authorized:
```bash
iflow login
```

### Error "API key empty"

Check that the file `~/.iflow/settings.json` contains a valid API key.

### Port Already in Use

Change the port in the `main.go` file (constant `PROXY_PORT`).

## License

MIT License

---

**Русская версия:** [README_RU.md](README_RU.md)