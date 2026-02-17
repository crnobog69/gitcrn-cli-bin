param(
  [string]$Prefix = "$env:LOCALAPPDATA\Programs\gitcrn",
  [string]$Version = "latest",
  [string]$ServerUrl = "https://100.91.132.35",
  [string]$Owner = "vltc",
  [string]$Repo = "gitcrn-cli",
  [switch]$Insecure
)

$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

& (Join-Path $ScriptDir "install.ps1") `
  -Prefix $Prefix `
  -Version $Version `
  -ServerUrl $ServerUrl `
  -Owner $Owner `
  -Repo $Repo `
  -Insecure:$Insecure
