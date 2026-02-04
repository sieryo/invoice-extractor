$ErrorActionPreference = "Stop"

$appName = "invoice-extractor"
$outDir  = "build/release"

Write-Host ">> Generating winres"
go-winres make -out ./cmd/invoice-extractor/invoice-extractor

Write-Host ">> Building exe"
go build -tags release -trimpath -ldflags "-s -w" `
  -o "$outDir\$appName.exe" `
  ./cmd/invoice-extractor

Write-Host ">> Signing exe"
signtool sign `
  /fd SHA256 `
  /n "Invoice Extractor" `
  "$outDir\$appName.exe"

Write-Host ">> Verifying signature (internal, self-signed)"

$signature = Get-AuthenticodeSignature "$outDir\$appName.exe"

if (-not $signature.SignerCertificate) {
    throw "Executable is NOT signed"
}

if ($signature.SignerCertificate.Subject -notlike "*Invoice Extractor*") {
    throw "Executable signed by unexpected certificate"
}

Write-Host "Signature present and signer verified (self-signed)"
Write-Host "Authenticode status: $($signature.Status)"



Write-Host ">> Generating checksum"
Get-FileHash "$outDir\$appName.exe" -Algorithm SHA256 |
  Out-File "$outDir\$appName.exe.sha256"

Write-Host ">> Release done"
