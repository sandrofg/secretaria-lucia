# Configura o Windows do laptop para manter o Ubuntu (WSL) e o DG Claw sempre no ar.
# Rodar DEPOIS de instalar o Ubuntu-24.04 e o DG Claw (passos 1 a 3 do roteiro).
# Uso: botao direito no PowerShell > "Executar como administrador", depois:
#   powershell -ExecutionPolicy Bypass -File .\configurar-servidor.ps1

$ErrorActionPreference = 'Stop'
$distro = 'Ubuntu-24.04'

Write-Host "`n[1/4] Conferindo o Ubuntu..." -ForegroundColor Cyan
$lista = (wsl.exe -l -q) -replace "`0", '' | Where-Object { $_.Trim() }
if ($lista -notcontains $distro) {
    throw "Nao achei o $distro no WSL. Rode antes: wsl --install -d $distro"
}
Write-Host "  ok: $distro instalado"

Write-Host "`n[2/4] Limitando o Ubuntu a 8 GB de RAM e ajustando a rede..." -ForegroundColor Cyan
$wslconfig = Join-Path $env:USERPROFILE '.wslconfig'
if (Test-Path $wslconfig) { Copy-Item $wslconfig "$wslconfig.bak" -Force; Write-Host "  copia do arquivo antigo: $wslconfig.bak" }
$build = [int](Get-CimInstance Win32_OperatingSystem).BuildNumber
$linhas = @('[wsl2]', 'memory=8GB', 'swap=4GB')
# Rede espelhada (Windows 11 22H2+): o painel e o Tailscale do Windows enxergam o Ubuntu direto.
if ($build -ge 22621) { $linhas += 'networkingMode=mirrored' } else { Write-Host "  Windows antigo (build $build): rede espelhada nao disponivel, seguindo sem ela." }
Set-Content -Path $wslconfig -Value $linhas -Encoding ascii
Write-Host "  ok: $wslconfig gravado"

Write-Host "`n[3/4] Ligando o systemd no Ubuntu (o DG Claw precisa dele)..." -ForegroundColor Cyan
wsl.exe -d $distro -u root -- sh -c "grep -q '^systemd=true' /etc/wsl.conf 2>/dev/null || printf '[boot]\nsystemd=true\n' >> /etc/wsl.conf"
Write-Host "  ok"

Write-Host "`n[4/4] Criando a tarefa que abre o Ubuntu sozinho ao entrar no Windows..." -ForegroundColor Cyan
$nome = 'DG Claw - manter Ubuntu ligado'
$acao = New-ScheduledTaskAction -Execute 'wsl.exe' -Argument "-d $distro --exec sleep infinity"
$gatilho = New-ScheduledTaskTrigger -AtLogOn -User $env:USERNAME
$config = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries `
    -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1) -Hidden
$quem = New-ScheduledTaskPrincipal -UserId $env:USERNAME -LogonType Interactive -RunLevel Limited
Register-ScheduledTask -TaskName $nome -Action $acao -Trigger $gatilho -Settings $config -Principal $quem -Force | Out-Null
Write-Host "  ok: tarefa '$nome' criada"

Write-Host "`nReiniciando o Ubuntu para aplicar..." -ForegroundColor Cyan
wsl.exe --shutdown
Start-Sleep -Seconds 3
Start-ScheduledTask -TaskName $nome
Start-Sleep -Seconds 15
Write-Host "`nStatus do DG Claw:" -ForegroundColor Cyan
wsl.exe -d $distro -- bash -lc "dgclaw status || echo 'dgclaw nao respondeu: abra o Ubuntu e rode dgclaw status'"
Write-Host "`nPronto. Falta so a parte de energia do roteiro (passo 5)." -ForegroundColor Green
