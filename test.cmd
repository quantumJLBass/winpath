@echo off
REM Test script for Windows PATH Optimizer
REM Run this from the project directory

echo Running Tests for Windows PATH Optimizer
echo ==========================================
echo.

echo Running unit tests...
go test -v ./...

if %ERRORLEVEL% NEQ 0 (
    echo.
    echo Tests FAILED
    exit /b 1
)

echo.
echo Tests PASSED
echo.

REM Run with coverage
echo Running with coverage...
go test -coverprofile=coverage.txt -covermode=atomic ./...

if exist coverage.txt (
    echo.
    echo Coverage summary:
    go tool cover -func=coverage.txt | findstr total:
)

echo.
echo Done!
