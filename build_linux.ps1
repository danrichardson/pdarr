Set-Location 'c:\src\_project-sqzarr'
New-Item -ItemType Directory -Force -Path dist | Out-Null
$env:GOOS = 'linux'
$env:GOARCH = 'amd64'

# Find go.exe
$goExe = $null
$candidates = @(
    'go',
    'C:\Program Files\Go\bin\go.exe',
    'C:\Go\bin\go.exe',
    "$env:USERPROFILE\go\bin\go.exe",
    "$env:USERPROFILE\AppData\Local\go\bin\go.exe",
    'C:\tools\go\bin\go.exe'
)
foreach ($candidate in $candidates) {
    try {
        $resolved = (Get-Command $candidate -ErrorAction Stop).Source
        $goExe = $resolved
        break
    } catch {}
}
if (-not $goExe) {
    Write-Error "go not found. Install Go from https://go.dev/dl/ and re-run."
    exit 1
}
Write-Host "Using go: $goExe"

$proc = Start-Process -FilePath $goExe `
    -ArgumentList 'build', '-trimpath', '-ldflags=-s -w', '-o', 'dist/sqzarr-linux-amd64', './cmd/sqzarr/' `
    -Wait -PassThru -NoNewWindow
Write-Host "Exit: $($proc.ExitCode)"
if ($proc.ExitCode -eq 0) {
    $size = (Get-Item 'dist\sqzarr-linux-amd64').Length
    Write-Host "Built: dist/sqzarr-linux-amd64 ($([math]::Round($size/1MB, 1)) MB)"
} else {
    exit 1
}
