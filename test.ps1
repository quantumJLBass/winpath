# Test script for Windows PATH Optimizer
# Run this from the project directory

param(
    [switch]$Coverage,
    [switch]$Verbose,
    [switch]$Race,
    [switch]$Bench,
    [string]$Package = "./..."
)

Write-Host "Running Tests for Windows PATH Optimizer" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan

# Build test arguments
$testArgs = @()

if ($Verbose) {
    $testArgs += "-v"
}

if ($Race) {
    $testArgs += "-race"
}

if ($Coverage) {
    $testArgs += "-coverprofile=coverage.txt"
    $testArgs += "-covermode=atomic"
}

# Run tests
Write-Host ""
Write-Host "Running unit tests..." -ForegroundColor Yellow

$testCmd = "go test $($testArgs -join ' ') $Package"
Write-Host "Command: $testCmd" -ForegroundColor DarkGray

Invoke-Expression $testCmd

if ($LASTEXITCODE -ne 0) {
    Write-Host ""
    Write-Host "Tests FAILED" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "Tests PASSED" -ForegroundColor Green

# Show coverage report if generated
if ($Coverage -and (Test-Path "coverage.txt")) {
    Write-Host ""
    Write-Host "Coverage Report:" -ForegroundColor Yellow
    go tool cover -func=coverage.txt | Select-Object -Last 1
    
    Write-Host ""
    Write-Host "To view detailed coverage:" -ForegroundColor Cyan
    Write-Host "  go tool cover -html=coverage.txt" -ForegroundColor White
}

# Run benchmarks if requested
if ($Bench) {
    Write-Host ""
    Write-Host "Running benchmarks..." -ForegroundColor Yellow
    go test -bench=. -benchmem $Package
}

Write-Host ""
Write-Host "Done!" -ForegroundColor Green
