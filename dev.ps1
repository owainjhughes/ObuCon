$backend = Start-Process -NoNewWindow -PassThru -FilePath "go" `
    -ArgumentList "run", "cmd/server/main.go" `
    -WorkingDirectory "$PSScriptRoot\backend"

$frontend = Start-Process -NoNewWindow -PassThru -FilePath "cmd.exe" `
    -ArgumentList "/c", "npm start" `
    -WorkingDirectory "$PSScriptRoot\frontend"

Write-Host "Dev servers started. Press Ctrl+C to stop." -ForegroundColor Cyan

try {
    while ($true) { Start-Sleep -Seconds 1 }
} finally {
    Write-Host "`nStopping dev servers..." -ForegroundColor Yellow
    Stop-Process -Id $backend.Id, $frontend.Id -ErrorAction SilentlyContinue
    Write-Host "Done." -ForegroundColor Green
}
