@echo off
cd /d c:\src\_project-sqzarr
if not exist dist mkdir dist
set GOOS=linux
set GOARCH=amd64
go build -trimpath -ldflags="-s -w" -o dist\sqzarr-linux-amd64 .\cmd\sqzarr\
if %ERRORLEVEL% EQU 0 (
    echo BUILD OK
    dir dist\sqzarr-linux-amd64
) else (
    echo BUILD FAILED: %ERRORLEVEL%
)
