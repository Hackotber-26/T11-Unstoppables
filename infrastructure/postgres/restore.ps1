[CmdletBinding(SupportsShouldProcess = $true, ConfirmImpact = "High")]
param(
    [Parameter(Mandatory = $true)]
    [string]$BackupPath
)

$ErrorActionPreference = "Stop"
$resolvedBackup = (Resolve-Path -LiteralPath $BackupPath).Path
$composeFile = Join-Path $PSScriptRoot "..\docker\docker-compose.yml"
$containerPath = "/tmp/trainsync-restore.dump"

if ($PSCmdlet.ShouldProcess("PostgreSQL database trainsync", "Restore from $resolvedBackup (existing data may be replaced)")) {
    docker compose -f $composeFile cp $resolvedBackup "postgres:$containerPath"
    if ($LASTEXITCODE -ne 0) {
        throw "copying the backup into the postgres container failed with exit code $LASTEXITCODE"
    }
    docker compose -f $composeFile exec -T postgres pg_restore --clean --if-exists --no-owner --username=postgres --dbname=trainsync $containerPath
    if ($LASTEXITCODE -ne 0) {
        throw "pg_restore failed with exit code $LASTEXITCODE"
    }
    docker compose -f $composeFile exec -T postgres rm -f $containerPath
    if ($LASTEXITCODE -ne 0) {
        Write-Warning "Restore succeeded, but the temporary dump could not be removed from the container."
    }
    Write-Output "Database restored from $resolvedBackup"
}