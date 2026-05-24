$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

$matches = Get-ChildItem -Recurse -File |
  Where-Object { $_.FullName -notmatch '\\.git\\|\\.cache\\|node_modules|dist|bin|obj' } |
  Select-String -Pattern '^(<<<<<<<\s|=======$|>>>>>>>\s)' -ErrorAction SilentlyContinue

if ($matches) {
  Write-Host "Git conflict markers found:" -ForegroundColor Red
  $matches | ForEach-Object {
    Write-Host "$($_.Path):$($_.LineNumber): $($_.Line)" -ForegroundColor Red
  }
  exit 1
}

Write-Host "No Git conflict markers found."
