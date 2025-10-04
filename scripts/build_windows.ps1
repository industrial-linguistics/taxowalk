Param(
    [string]$Version
)

if (-not $Version) {
    $Version = Get-Content -Path (Join-Path $PSScriptRoot '..' 'VERSION') -Raw
    $Version = $Version.Trim()
}

$Root = Resolve-Path (Join-Path $PSScriptRoot '..')
$Dist = Join-Path $Root 'dist/windows'
$Build = Join-Path $Root 'build/windows'

Remove-Item -Recurse -Force -ErrorAction SilentlyContinue $Build
New-Item -ItemType Directory -Force -Path $Build, $Dist | Out-Null

$env:GOOS = 'windows'
$env:GOARCH = 'amd64'
$env:CGO_ENABLED = '0'

$binary = Join-Path $Build 'taxowalk.exe'
go build -trimpath -ldflags "-s -w -X main.version=$Version" -o $binary (Join-Path $Root 'cmd/taxowalk')

$helpSource = Join-Path $Root 'docs/taxowalk-help.txt'
$zipPath = Join-Path $Dist ("taxowalk_{0}_windows_amd64.zip" -f $Version)

Compress-Archive -Path $binary, $helpSource -DestinationPath $zipPath -Force

Write-Output "Built $zipPath"
