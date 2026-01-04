@echo off
echo Building Windows PATH Optimizer...
echo.

echo Downloading dependencies...
go mod tidy
if errorlevel 1 (
    echo Failed to download dependencies
    exit /b 1
)

echo Building executable...
go build -ldflags="-s -w" -o WinPath.exe .
if errorlevel 1 (
    echo Build failed
    exit /b 1
)

echo.
echo Build complete!
echo Run with: WinPath.exe
echo.
pause
