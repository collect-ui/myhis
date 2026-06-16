@echo off
setlocal

echo Building main.exe...
go build -ldflags="-s -w" -o windows/main.exe -v main.go
if errorlevel 1 (
    echo Build failed!
    pause
    exit /b 1
)

echo Compressing with UPX...
F:\upx-5.1.0-win64\upx.exe windows/main.exe
if errorlevel 1 (
    echo Compression failed!
    pause
    exit /b 1
)

echo Done!
dir windows\main.exe
pause
