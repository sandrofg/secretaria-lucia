# Envia uma mensagem do Argos para o Sandro no Telegram (canal de saida das rotinas).
# Uso: powershell -File enviar-telegram.ps1 -Arquivo mensagem.txt   (ou -Texto "...")
param([string]$Texto, [string]$Arquivo)
$ErrorActionPreference = 'Stop'
if ($Arquivo) { $Texto = Get-Content $Arquivo -Raw -Encoding UTF8 }
if (-not $Texto) { throw 'Nada para enviar.' }
$tokenLine = Get-Content "$env:USERPROFILE\.claude\channels\lucia-token.env" | Where-Object { $_ -match '^TELEGRAM_BOT_TOKEN=' }
$token = ($tokenLine -replace '^TELEGRAM_BOT_TOKEN=', '').Trim().Trim('"')
$access = Get-Content "$env:USERPROFILE\.claude\channels\telegram\access.json" -Raw | ConvertFrom-Json
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
foreach ($chat in $access.allowFrom) {
  $body = @{ chat_id = $chat; text = $Texto } | ConvertTo-Json -Compress
  $bytes = [Text.Encoding]::UTF8.GetBytes($body)
  $r = Invoke-RestMethod -Method Post -Uri "https://api.telegram.org/bot$token/sendMessage" -ContentType 'application/json; charset=utf-8' -Body $bytes
  if ($r.ok) { "enviado" } else { "falhou" }
}
