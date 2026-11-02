param(
    [string]$OutputDirectory
)

$ErrorActionPreference = "Stop"
$composeFile = Join-Path $PSScriptRoot "..\docker\docker-compose.yml"
if ([string]::IsNullOrWhiteSpace($OutputDirectory)) {
    $OutputDirectory = Join-Path $PSScriptRoot "backups"
}
$resolvedOutput = [System.IO.Path]::GetFullPath($OutputDirectory)
New-Item -ItemType Directory -Force -Path $resolvedOutput | Out-Null

$fileName = "trainsync-$(Get-Date -Format 'yyyyMMdd-HHmmss').dump"
$containerPath = "/tmp/$fileName"
$localPath = Join-Path $resolvedOutput $fileName

docker compose -f $composeFile exec -T postgres pg_dump -U postgres -d trainsync -Fc -f $containerPath
if ($LASTEXITCODE -ne 0) {
    throw "pg_dump failed with exit code $LASTEXITCODE"
}
docker compose -f $composeFile cp "postgres:$containerPath" $localPath
if ($LASTEXITCODE -ne 0) {
    throw "copying the database dump failed with exit code $LASTEXITCODE"
}
docker compose -f $composeFile exec -T postgres rm -f $containerPath
if ($LASTEXITCODE -ne 0) {
    Write-Warning "Backup succeeded, but the temporary dump could not be removed from the container."
}
Write-Output "Database backup written to $localPath"