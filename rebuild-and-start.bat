@echo off
title iFlow Proxy for GLM-5
echo Starting iFlow Proxy for GLM-5...
echo.
taskkill /F /IM iflow-proxy.exe 2>nul
go build -o iflow-proxy.exe main.go
iflow-proxy.exe
echo.
echo Proxy stopped.
pause
