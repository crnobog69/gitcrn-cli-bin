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
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

& (Join-Path $ScriptDir "install.ps1") `
  -Prefix $Prefix `
  -Version $Version `
  -Provider $Provider `
  -ServerUrl $ServerUrl `
  -ApiUrl $ApiUrl `
  -Owner $Owner `
  -Repo $Repo `
  -Token $Token `
  -Insecure:$Insecure
