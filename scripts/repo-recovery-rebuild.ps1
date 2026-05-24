<#
.SYNOPSIS
  Safely prepares the G2S_MC repo for the hybrid rebuild.

.DESCRIPTION
  This script does not push anything.
  It:
    1. Tags the current HEAD as an archived unsafe/pre-rebuild state.
    2. Creates/switches to a cleanup branch from a cleaner base ref.
    3. Adds the rebuild project-definition/guardrails document.
    4. Adds a conflict-marker scan script.
    5. Updates GitHub Actions CI to run the scan and Go tests.
    6. Runs the scan and tests locally.
    7. Commits the guardrail files if verification passes.
    8. Creates a rebuild branch from that green cleanup commit.

.PARAMETER DocSource
  Path to G2S_MC_REBUILD_PROJECT_DEFINITION_AND_GUARDRAILS.md downloaded from ChatGPT.

.PARAMETER BaseRef
  Base ref for the cleanup branch. Default is HEAD^, assuming current HEAD is the dirty "Apply local working changes" commit.

.PARAMETER FromCurrentHead
  Use current HEAD instead of BaseRef. Not recommended unless you have already fixed conflict markers.

.PARAMETER SkipTests
  Skip local go test ./... verification.

.EXAMPLE
  .\scripts\repo-recovery-rebuild.ps1 -DocSource "$env:USERPROFILE\Downloads\G2S_MC_REBUILD_PROJECT_DEFINITION_AND_GUARDRAILS_v2.md"

.EXAMPLE
  .\scripts\repo-recovery-rebuild.ps1 -DocSource ".\G2S_MC_REBUILD_PROJECT_DEFINITION_AND_GUARDRAILS_v2.md" -BaseRef d43378f080e3b7a5dc5ed6b961f5311036af10de
#>

param(
    [string]$DocSource = "",
    [string]$CleanupBranch = "cleanup/restore-build",
    [string]$RebuildBranch = "rebuild/input-action-engine",
    [string]$BaseRef = "HEAD^",
    [switch]$FromCurrentHead,
    [switch]$SkipTests
)

$ErrorActionPreference = "Stop"

function Invoke-Checked {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Command,

        [Parameter(ValueFromRemainingArguments = $true)]
        [string[]]$Arguments
    )

    & $Command @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "Command failed with exit code ${LASTEXITCODE}: $Command $($Arguments -join ' ')"
    }
}

function Require-Command {
    param([string]$Name)

    $cmd = Get-Command $Name -ErrorAction SilentlyContinue
    if (-not $cmd) {
        throw "Required command '$Name' was not found in PATH."
    }
}

Require-Command git

$repoRoot = (& git rev-parse --show-toplevel).Trim()
if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($repoRoot)) {
    throw "This script must be run inside the G2S_MC git repository."
}
Set-Location $repoRoot

$currentBranch = (& git rev-parse --abbrev-ref HEAD).Trim()
$currentHead = (& git rev-parse --short HEAD).Trim()
Write-Host "Repo: $repoRoot"
Write-Host "Current branch: $currentBranch"
Write-Host "Current HEAD: $currentHead"

$status = (& git status --porcelain)
if ($status) {
    Write-Host ""
    Write-Host "Working tree has uncommitted changes:" -ForegroundColor Yellow
    $status | ForEach-Object { Write-Host "  $_" -ForegroundColor Yellow }
    throw "Please commit, stash, or discard local changes before running this recovery script."
}

if ([string]::IsNullOrWhiteSpace($DocSource)) {
    throw "Pass -DocSource with the path to the downloaded project-definition Markdown file."
}

$resolvedDocSource = Resolve-Path $DocSource -ErrorAction Stop
if (-not (Test-Path $resolvedDocSource)) {
    throw "DocSource not found: $DocSource"
}

$stamp = Get-Date -Format "yyyyMMdd-HHmmss"
$archiveTag = "archive/broken-main-before-rebuild-$stamp"

Write-Host ""
Write-Host "Creating archive tag: $archiveTag"
Invoke-Checked git tag $archiveTag HEAD

$baseToUse = $BaseRef
if ($FromCurrentHead) {
    $baseToUse = "HEAD"
}
Write-Host "Cleanup branch base: $baseToUse"

$existingCleanup = @(& git branch --list $CleanupBranch) -join ""
if ($existingCleanup) {
    Write-Host "Switching to existing branch: $CleanupBranch"
    Invoke-Checked git switch $CleanupBranch
} else {
    Write-Host "Creating cleanup branch: $CleanupBranch"
    Invoke-Checked git switch -c $CleanupBranch $baseToUse
}

# Create project-definition doc.
$projectDefDir = Join-Path $repoRoot "docs\project-definition"
New-Item -ItemType Directory -Force -Path $projectDefDir | Out-Null
$projectDefTarget = Join-Path $projectDefDir "G2S_MC_REBUILD_PROJECT_DEFINITION_AND_GUARDRAILS.md"
Copy-Item -Force $resolvedDocSource $projectDefTarget
Write-Host "Added project-definition doc: $projectDefTarget"

# Add conflict-marker check script.
$scriptsDir = Join-Path $repoRoot "scripts"
New-Item -ItemType Directory -Force -Path $scriptsDir | Out-Null
$conflictScript = Join-Path $scriptsDir "check-conflict-markers.ps1"

@'
$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

$skipPathPattern = '\\.git\\|\\.cache\\|\\node_modules\\|\\dist\\|\\bin\\|\\obj\\'
$textExtensions = @(
    ".go", ".mod", ".sum", ".ps1", ".psm1", ".json", ".md", ".yml", ".yaml",
    ".html", ".css", ".js", ".ts", ".tsx", ".xml", ".xsd", ".sql", ".txt", ".service"
)

$files = Get-ChildItem -Path $repoRoot -Recurse -File |
    Where-Object { $_.FullName -notmatch $skipPathPattern } |
    Where-Object { $textExtensions -contains $_.Extension.ToLowerInvariant() }

$pattern = '^(<<<<<<<\s|=======$|>>>>>>>\s)'

$matches = @()
foreach ($file in $files) {
    try {
        $found = Select-String -Path $file.FullName -Pattern $pattern -ErrorAction Stop
        if ($found) {
            $matches += $found
        }
    } catch {
        Write-Warning "Could not scan $($file.FullName): $($_.Exception.Message)"
    }
}

if ($matches.Count -gt 0) {
    Write-Host "Git conflict markers found:" -ForegroundColor Red
    foreach ($match in $matches) {
        $relative = Resolve-Path -Relative $match.Path
        Write-Host ("{0}:{1}: {2}" -f $relative, $match.LineNumber, $match.Line) -ForegroundColor Red
    }
    exit 1
}

Write-Host "No Git conflict markers found."
'@ | Set-Content -Path $conflictScript -Encoding UTF8

Write-Host "Added conflict-marker scan: $conflictScript"

# Update CI workflow.
$workflowDir = Join-Path $repoRoot ".github\workflows"
New-Item -ItemType Directory -Force -Path $workflowDir | Out-Null
$ciPath = Join-Path $workflowDir "ci.yml"

@'
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Check out
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - name: Check conflict markers
        shell: pwsh
        run: ./scripts/check-conflict-markers.ps1

      - name: Test
        run: go test ./...
'@ | Set-Content -Path $ciPath -Encoding UTF8

Write-Host "Updated CI workflow: $ciPath"

Write-Host ""
Write-Host "Running conflict-marker scan..."
Invoke-Checked pwsh -NoProfile -ExecutionPolicy Bypass -File $conflictScript

if (-not $SkipTests) {
    Require-Command go
    Write-Host ""
    Write-Host "Running go test ./..."
    Invoke-Checked go test ./...
} else {
    Write-Host "Skipping go test ./... because -SkipTests was supplied." -ForegroundColor Yellow
}

Write-Host ""
Write-Host "Staging guardrail files..."
Invoke-Checked git add `
    "docs/project-definition/G2S_MC_REBUILD_PROJECT_DEFINITION_AND_GUARDRAILS.md" `
    "scripts/check-conflict-markers.ps1" `
    ".github/workflows/ci.yml"

$pending = (& git diff --cached --name-only).Trim()
if (-not $pending) {
    Write-Host "No staged changes to commit."
} else {
    Write-Host "Committing guardrail files..."
    Invoke-Checked git commit -m "Add rebuild project definition and repo guardrails"
}

$existingRebuild = @(& git branch --list $RebuildBranch) -join ""
if ($existingRebuild) {
    Write-Host "Rebuild branch already exists: $RebuildBranch" -ForegroundColor Yellow
    Write-Host "Switching to it."
    Invoke-Checked git switch $RebuildBranch
} else {
    Write-Host "Creating rebuild branch: $RebuildBranch"
    Invoke-Checked git switch -c $RebuildBranch
}

Write-Host ""
Write-Host "Done." -ForegroundColor Green
Write-Host "Archive tag: $archiveTag"
Write-Host "Cleanup branch: $CleanupBranch"
Write-Host "Rebuild branch: $RebuildBranch"
Write-Host ""
Write-Host "Next manual commands when ready:"
Write-Host "  git push origin $archiveTag"
Write-Host "  git push -u origin $CleanupBranch"
Write-Host "  git push -u origin $RebuildBranch"

