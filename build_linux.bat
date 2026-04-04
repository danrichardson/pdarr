@echo off
cd /d c:\src\_project-pdarr
if not exist dist mkdir dist
set GOOS=linux
set GOARCH=amd64
go build -trimpath -ldflags="-s -w" -o dist\pdarr-linux-amd64 .\cmd\pdarr\
if %ERRORLEVEL% EQU 0 (
    echo BUILD OK
    dir dist\pdarr-linux-amd64
) else (
    echo BUILD FAILED: %ERRORLEVEL%
)
