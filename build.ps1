# Build script for Windows PATH Optimizer
# Run this from the project directory

Write-Host "Building Windows PATH Optimizer..." -ForegroundColor Cyan

# Download dependencies
Write-Host "Downloading dependencies..." -ForegroundColor Yellow
go mod tidy
if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to download dependencies" -ForegroundColor Red
    exit 1
}

# Build with optimizations
Write-Host "Building executable..." -ForegroundColor Yellow
go build -ldflags="-s -w" -o WinPath.exe .
if ($LASTEXITCODE -ne 0) {
    Write-Host "Build failed" -ForegroundColor Red
    exit 1
}

$size = (Get-Item WinPath.exe).Length / 1MB
Write-Host "Build complete! Size: $([math]::Round($size, 2)) MB" -ForegroundColor Green
Write-Host ""
Write-Host "Run with: .\WinPath.exe" -ForegroundColor Cyan
Write-Host "Run as admin: Start-Process -Verb RunAs .\WinPath.exe" -ForegroundColor Cyan
