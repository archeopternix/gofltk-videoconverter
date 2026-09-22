[CmdletBinding()]
param(
    [Parameter()]
    [string]$ToolsDirectory
)

$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$distDirectory = Join-Path $repositoryRoot 'dist'
$bundleDirectory = Join-Path $distDirectory 'videoconverter'
$archivePath = Join-Path $distDirectory 'videoconverter.zip'
$resolvedTools = $null

if (-not [string]::IsNullOrWhiteSpace($ToolsDirectory)) {
    $resolvedTools = (Resolve-Path -LiteralPath $ToolsDirectory).Path
    $requiredFiles = @(
        'ffmpeg/bin/ffmpeg.exe',
        'ffmpeg/bin/ffprobe.exe',
        'VirtualDub2/vdub64.exe'
    )
    foreach ($requiredFile in $requiredFiles) {
        $requiredPath = Join-Path $resolvedTools $requiredFile
        if (-not (Test-Path -LiteralPath $requiredPath -PathType Leaf)) {
            throw "Tools directory is incomplete; missing file: $requiredPath"
        }
    }
    $aviSynthPlugins = Join-Path $resolvedTools 'AviSynth+/plugins64+'
    if (-not (Test-Path -LiteralPath $aviSynthPlugins -PathType Container)) {
        throw "Tools directory is incomplete; missing directory: $aviSynthPlugins"
    }
}

New-Item -ItemType Directory -Path $distDirectory -Force | Out-Null

$resolvedDist = [System.IO.Path]::GetFullPath($distDirectory)
$resolvedBundle = [System.IO.Path]::GetFullPath($bundleDirectory)
if (-not $resolvedBundle.StartsWith($resolvedDist + [System.IO.Path]::DirectorySeparatorChar, [System.StringComparison]::OrdinalIgnoreCase)) {
    throw "Refusing to replace bundle outside dist: $resolvedBundle"
}

if (Test-Path -LiteralPath $bundleDirectory) {
    Remove-Item -LiteralPath $bundleDirectory -Recurse -Force
}
if (Test-Path -LiteralPath $archivePath) {
    Remove-Item -LiteralPath $archivePath -Force
}

New-Item -ItemType Directory -Path (Join-Path $bundleDirectory 'config') -Force | Out-Null

$env:GOOS = 'windows'
$env:GOARCH = 'amd64'
$env:CGO_ENABLED = '0'
$binaryPath = Join-Path $bundleDirectory 'videoconverter.exe'
Push-Location $repositoryRoot
try {
    & go build -trimpath -o $binaryPath ./cmd/medialang
    if ($LASTEXITCODE -ne 0) {
        throw "go build failed with exit code $LASTEXITCODE"
    }
}
finally {
    Pop-Location
}

Copy-Item -LiteralPath (Join-Path $repositoryRoot 'cmd/medialang/vdub.bat') -Destination $bundleDirectory
Copy-Item -LiteralPath (Join-Path $repositoryRoot 'config/workflows') -Destination (Join-Path $bundleDirectory 'config') -Recurse

if ([string]::IsNullOrWhiteSpace($ToolsDirectory)) {
    Copy-Item -LiteralPath (Join-Path $repositoryRoot 'config/app.windows.yaml') -Destination (Join-Path $bundleDirectory 'config/app.windows.yaml')
    Write-Host 'No tools directory supplied; tool locations remain configurable in config/app.windows.yaml.'
}
else {
    $bundledTools = Join-Path $bundleDirectory 'tools'
    New-Item -ItemType Directory -Path $bundledTools -Force | Out-Null
    Get-ChildItem -LiteralPath $resolvedTools -Force | Copy-Item -Destination $bundledTools -Recurse -Force
    Copy-Item -LiteralPath (Join-Path $PSScriptRoot 'app.windows.bundled.yaml') -Destination (Join-Path $bundleDirectory 'config/app.windows.yaml')
    Write-Host "Bundled tools from $resolvedTools and linked them from config/app.windows.yaml."
}

& $binaryPath -help
if ($LASTEXITCODE -ne 0) {
    throw "Packaged executable help check failed with exit code $LASTEXITCODE"
}

Compress-Archive -LiteralPath $bundleDirectory -DestinationPath $archivePath -CompressionLevel Optimal
Write-Host "Windows bundle created: $archivePath"
