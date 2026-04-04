@echo off
cd /d c:\src\_project-pdarr
if not exist dist mkdir dist

set GOOS=linux
set GOARCH=amd64
go build -trimpath -ldflags="-s -w" -o dist\pdarr-linux-amd64 .\cmd\pdarr\
if %ERRORLEVEL% NEQ 0 (
    echo BUILD FAILED
    exit /b 1
)
echo BUILD OK > c:\src\_project-pdarr\build_result.txt
dir dist\pdarr-linux-amd64 >> c:\src\_project-pdarr\build_result.txt
