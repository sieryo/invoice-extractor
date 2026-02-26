param(
    [string]$Version = "dev",
    [switch]$SkipSign
)

$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$backendDir = Resolve-Path (Join-Path $scriptDir "..")
Set-Location $backendDir

$appName = "invoice-extractor"
$certName = "Invoice Extractor"
$outDir = Join-Path $backendDir "build/release"
$exePath = Join-Path $outDir "$appName.exe"
$repoRoot = Resolve-Path (Join-Path $backendDir "..")
$versionFilePath = Join-Path $repoRoot "Version.txt"

if ($Version -eq "dev" -and (Test-Path $versionFilePath)) {
    $versionFromFile = (Get-Content $versionFilePath | Select-Object -First 1).Trim()
    if ($versionFromFile) {
        $Version = $versionFromFile
    }
}

if (Test-Path $outDir) {
    Remove-Item $outDir -Recurse -Force
}
New-Item -ItemType Directory -Path $outDir -Force | Out-Null

Write-Host ">> Generating winres"
go-winres make -in ./winres/winres.json -out ./cmd/invoice-extractor/invoice-extractor

Write-Host ">> Building exe (version=$Version)"
go build -tags release -trimpath -ldflags "-s -w -X main.version=$Version" `
  -o $exePath `
  ./cmd/invoice-extractor

Write-Host ">> Copying tools/pdftotext/bin (if present)"
$toolsBinSourceRepo = Join-Path $repoRoot "tools/pdftotext/bin"
$toolsBinSourceBackend = Join-Path $backendDir "tools/pdftotext/bin"
$toolsBinSource = $null
if (Test-Path $toolsBinSourceRepo) {
    $toolsBinSource = $toolsBinSourceRepo
} elseif (Test-Path $toolsBinSourceBackend) {
    $toolsBinSource = $toolsBinSourceBackend
}

if ($toolsBinSource) {
    $toolsBinTarget = Join-Path $outDir "tools/pdftotext/bin"
    New-Item -ItemType Directory -Path $toolsBinTarget -Force | Out-Null
    Copy-Item (Join-Path $toolsBinSource "*") $toolsBinTarget -Recurse -Force
} else {
    Write-Warning "tools/pdftotext/bin not found; ensure pdftotext runtime files are bundled"
}

Write-Host ">> Copying Version.txt (if present)"
if (Test-Path $versionFilePath) {
    Copy-Item $versionFilePath (Join-Path $outDir "Version.txt") -Force
} else {
    Write-Warning "Version.txt not found at repo root"
}

if (-not $SkipSign) {
    $signTool = Get-Command signtool -ErrorAction SilentlyContinue
    if ($null -ne $signTool) {
        Write-Host ">> Signing exe"
        signtool sign `
          /fd SHA256 `
          /n $certName `
          $exePath

        Write-Host ">> Verifying signature"
        $signature = Get-AuthenticodeSignature $exePath

        if (-not $signature.SignerCertificate) {
            throw "Executable is NOT signed"
        }

        if ($signature.SignerCertificate.Subject -notlike "*$certName*") {
            throw "Executable signed by unexpected certificate"
        }

        Write-Host "Signature present and signer verified"
        Write-Host "Authenticode status: $($signature.Status)"
    } else {
        Write-Warning "signtool not found; skipping signing"
    }
} else {
    Write-Host ">> Skip signing requested"
}

Write-Host ">> Creating zip package"
$zipPath = Join-Path $backendDir "build/$appName-$Version.zip"
if (Test-Path $zipPath) {
    Remove-Item $zipPath -Force
}
Compress-Archive -Path (Join-Path $outDir "*") -DestinationPath $zipPath -CompressionLevel Optimal

Write-Host ">> Uploading zip to Google Drive"
$driveFolder = "invoice-extractor"
$saveDrive = Get-Command save-drive -ErrorAction SilentlyContinue
if ($null -eq $saveDrive) {
    throw "save-drive command not found. Install or add it to PATH."
}

& save-drive push --file $zipPath --to $driveFolder --mkdir --replace
if ($LASTEXITCODE -ne 0) {
    throw "save-drive push failed with exit code $LASTEXITCODE"
}

Write-Host ">> Generating checksum"
Get-FileHash $exePath -Algorithm SHA256 |
  Out-File (Join-Path $outDir "$appName.exe.sha256")

Write-Host ">> Release done: $outDir"
Write-Host ">> Zip package: $zipPath"
