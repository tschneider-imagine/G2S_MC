param(
    [switch]$Run
)

$ErrorActionPreference = "Stop"

function Require-Command {
    param([string]$Name)

    $command = Get-Command $Name -ErrorAction SilentlyContinue
    if ($command) {
        return $command.Source
    }
    return $null
}

$go = Require-Command "go"
if (-not $go) {
    $defaultGo = "C:\Program Files\Go\bin\go.exe"
    if (Test-Path $defaultGo) {
        $go = $defaultGo
    }
}

if (-not $go) {
    throw "Required command 'go' was not found in PATH or at C:\Program Files\Go\bin\go.exe."
}

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$env:GOCACHE = Join-Path $repoRoot ".cache\go-build"
$env:GOMODCACHE = Join-Path $repoRoot ".cache\gomod"

& $go test ./...

if ($Run) {
    & $go run ./cmd/g2s-mute -config ./configs/config.example.json -simulate-trigger
}
