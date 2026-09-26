$ErrorActionPreference = "Stop"

New-Item -ItemType Directory -Force -Path "dist" | Out-Null

Write-Host "Building Windows..."
$env:GOOS="windows"
$env:GOARCH="amd64"
$env:CGO_ENABLED="0"
go build -o dist\ImgCrypt-windows-amd64.exe .

Write-Host "Building macOS Intel..."
$env:GOOS="darwin"
$env:GOARCH="amd64"
$env:CGO_ENABLED="0"
go build -o dist\ImgCrypt-macos-amd64 .

Write-Host "Building macOS Apple Silicon..."
$env:GOOS="darwin"
$env:GOARCH="arm64"
$env:CGO_ENABLED="0"
go build -o dist\ImgCrypt-macos-arm64 .

Write-Host "Building Linux x64..."
$env:GOOS="linux"
$env:GOARCH="amd64"
$env:CGO_ENABLED="0"
go build -o dist\ImgCrypt-linux-amd64 .

Write-Host "Building Linux ARM64..."
$env:GOOS="linux"
$env:GOARCH="arm64"
$env:CGO_ENABLED="0"
go build -o dist\ImgCrypt-linux-arm64 .

Write-Host ""
Write-Host "Build complete."