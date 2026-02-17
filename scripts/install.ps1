param(
  [string]$Prefix = "$env:LOCALAPPDATA\Programs\gitcrn",
  [string]$Version = "latest",
  [string]$ServerUrl = "http://100.91.132.35:5000",
  [string]$Owner = "vltc",
  [string]$Repo = "gitcrn-cli",
  [string]$Token = $env:GITEA_TOKEN,
  [switch]$Insecure
)

$ErrorActionPreference = "Stop"

function Resolve-Arch {
  switch ($env:PROCESSOR_ARCHITECTURE.ToUpperInvariant()) {
    "AMD64" { return "amd64" }
    "ARM64" { return "arm64" }
    default { throw "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE" }
  }
}

if ($Insecure) {
  [System.Net.ServicePointManager]::ServerCertificateValidationCallback = { $true }
}

if ([string]::IsNullOrWhiteSpace($ServerUrl) -or
    [string]::IsNullOrWhiteSpace($Owner) -or
    [string]::IsNullOrWhiteSpace($Repo)) {
  throw "ServerUrl, Owner and Repo are required."
}

$headers = @{}
if (-not [string]::IsNullOrWhiteSpace($Token)) {
  $headers["Authorization"] = "token $Token"
}

$os = "windows"
$arch = Resolve-Arch

if ($Version -eq "latest") {
  $apiUrl = "{0}/api/v1/repos/{1}/{2}/releases/latest" -f $ServerUrl.TrimEnd('/'), $Owner, $Repo
  $release = Invoke-RestMethod -Uri $apiUrl -Headers $headers
  if (-not $release.tag_name) {
    throw "Could not resolve latest release tag from $apiUrl"
  }
  $Version = [string]$release.tag_name
}

$asset = "gitcrn-$os-$arch.exe"
$downloadUrl = "{0}/{1}/{2}/releases/download/{3}/{4}" -f $ServerUrl.TrimEnd('/'), $Owner, $Repo, $Version, $asset

New-Item -ItemType Directory -Force -Path $Prefix | Out-Null
$Target = Join-Path $Prefix "gitcrn.exe"
$TempFile = [System.IO.Path]::GetTempFileName()

try {
  Write-Host "Downloading $downloadUrl ..."
  Invoke-WebRequest -Uri $downloadUrl -Headers $headers -OutFile $TempFile
  Move-Item -Force -Path $TempFile -Destination $Target
} finally {
  if (Test-Path $TempFile) {
    Remove-Item -Force $TempFile
  }
}

Write-Host "Installed: $Target"

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
$segments = @()
if ($userPath) {
  $segments = $userPath -split ";"
}

if (-not ($segments -contains $Prefix)) {
  $newPath = if ([string]::IsNullOrWhiteSpace($userPath)) { $Prefix } else { "$userPath;$Prefix" }
  [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
  Write-Host "Added to user PATH: $Prefix"
  Write-Host "Restart terminal to use 'gitcrn'."
}
