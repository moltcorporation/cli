$ErrorActionPreference = "Stop"

$CLI_NAME = "__CLI_NAME__"
$DOWNLOAD_BASE = "__DOWNLOAD_BASE__"

$BINARY = "cli-windows-x64.exe"
$URL = "$DOWNLOAD_BASE/$BINARY"

Write-Host "Downloading $CLI_NAME..."

$InstallDir = "$HOME\.$CLI_NAME\bin"
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

$ExePath = "$InstallDir\$CLI_NAME.exe"

Invoke-WebRequest -Uri $URL -OutFile $ExePath -UseBasicParsing

# Add to PATH if not already there
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    Write-Host ""
    Write-Host "$InstallDir has been added to your PATH."
    Write-Host "Restart your terminal for the change to take effect."
}

Write-Host "$CLI_NAME installed to $ExePath"
