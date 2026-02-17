param(
  [string]$Prefix = "$env:LOCALAPPDATA\Programs\gitcrn",
  [string]$Version = "latest",
  [ValidateSet("github", "gitea")]
  [string]$Provider = "github",
  [string]$ServerUrl = "https://github.com",
  [string]$ApiUrl = "https://api.github.com",
  [string]$Owner = "crnobog69",
  [string]$Repo = "gitcrn-cli-bin",
  [string]$Token = $(if ($env:GITCRN_TOKEN) { $env:GITCRN_TOKEN } else { $env:GITEA_TOKEN }),
  [switch]$Insecure
)

$ErrorActionPreference = "Stop"
$InstallScriptUrl = if ($env:GITCRN_INSTALL_PS1_URL) {
  $env:GITCRN_INSTALL_PS1_URL
} else {
  "https://raw.githubusercontent.com/crnobog69/gitcrn-cli-bin/refs/heads/master/scripts/install.ps1"
}

$localInstall = $null
if ($MyInvocation.MyCommand.Path) {
  $scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
  $candidate = Join-Path $scriptDir "install.ps1"
  if (Test-Path $candidate) {
    $localInstall = $candidate
  }
}

if ($localInstall) {
  & $localInstall `
    -Prefix $Prefix `
    -Version $Version `
    -Provider $Provider `
    -ServerUrl $ServerUrl `
    -ApiUrl $ApiUrl `
    -Owner $Owner `
    -Repo $Repo `
    -Token $Token `
    -Insecure:$Insecure
  exit 0
}

$tempInstall = Join-Path ([System.IO.Path]::GetTempPath()) ("gitcrn-install-" + [guid]::NewGuid().ToString() + ".ps1")
try {
  Invoke-WebRequest -Uri $InstallScriptUrl -OutFile $tempInstall
  & $tempInstall `
    -Prefix $Prefix `
    -Version $Version `
    -Provider $Provider `
    -ServerUrl $ServerUrl `
    -ApiUrl $ApiUrl `
    -Owner $Owner `
    -Repo $Repo `
    -Token $Token `
    -Insecure:$Insecure
} finally {
  if (Test-Path $tempInstall) {
    Remove-Item -Force $tempInstall
  }
}
