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

$binaries = @('taxowalk', 'taxoname', 'taxopath')
$builtPaths = @()
foreach ($name in $binaries) {
    $binaryPath = Join-Path $Build ("{0}.exe" -f $name)
    go build -trimpath -ldflags "-s -w -X main.version=$Version" -o $binaryPath (Join-Path $Root 'cmd' $name)
    $builtPaths += $binaryPath
}

$helpFiles = @('taxowalk-help.txt', 'taxoname-help.txt', 'taxopath-help.txt') |
    ForEach-Object { Join-Path $Root 'docs' $_ }

$zipPath = Join-Path $Dist ("taxowalk_{0}_windows_amd64.zip" -f $Version)
Compress-Archive -Path ($builtPaths + $helpFiles) -DestinationPath $zipPath -Force

Write-Output "Built $zipPath"
