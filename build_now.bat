@echo off
cd /d c:\src\_project-sqzarr
if not exist dist mkdir dist
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0
go build -trimpath -ldflags="-s -w" -o dist\sqzarr-linux-amd64 .\cmd\sqzarr\
if %ERRORLEVEL% EQU 0 (
    echo BUILD_OK
    dir dist\sqzarr-linux-amd64
) else (
    echo BUILD_FAILED %ERRORLEVEL%
)
