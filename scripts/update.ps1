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
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

& (Join-Path $ScriptDir "install.ps1") `
  -Prefix $Prefix `
  -Version $Version `
  -ServerUrl $ServerUrl `
  -Owner $Owner `
  -Repo $Repo `
  -Token $Token `
  -Insecure:$Insecure
