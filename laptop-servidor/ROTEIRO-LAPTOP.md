# Roteiro: Legion 5 como servidor do DG Claw (sem apagar o Windows)

Tempo total: cerca de 1 hora. Leve esta pasta inteira para o laptop (pendrive ou Google Drive).

O que vai acontecer: o Ubuntu roda dentro do Windows (WSL). O DG Claw roda nele, por trás,
enquanto você usa o Windows normalmente. O laptop precisa ficar ligado, na tomada e com seu
usuário entrado (tela bloqueada não tem problema).

---

## Passo 1. Instalar o Ubuntu 24.04 dentro do Windows

1. Clique no Iniciar, digite **PowerShell**, clique com o botão direito em **Windows PowerShell**
   e escolha **Executar como administrador**.
2. Cole (botão direito) e aperte Enter:

   ```
   wsl --install -d Ubuntu-24.04
   ```

3. Quando terminar, **reinicie o laptop**.
4. Ao voltar, abre sozinha uma janela do Ubuntu. Se não abrir, Iniciar → digite **Ubuntu** → abra.
5. Ele pede um **nome de usuário** (minúsculo, sem espaço, ex.: `sandro`) e uma **senha**.
   Ao digitar a senha nada aparece na tela: é normal. Anote os dois.

> Se der erro "virtualização desabilitada": reinicie, entre na BIOS (no Legion, aperte F2 ao
> ligar), procure **Intel Virtualization Technology** ou **AMD SVM**, ative, salve (F10).

## Passo 2. Instalar e logar o Claude Code no Ubuntu

Na janela do Ubuntu, um comando de cada vez (cole com o botão direito):

```
curl -fsSL https://claude.ai/install.sh | bash
```

```
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc && source ~/.bashrc
```

Depois digite `claude`, Enter, e faça o login exatamente como na Etapa 3 do guia da imersão
(opção 1, conta de assinatura; nunca a API).

## Passo 3. Instalar o DG Claw

Siga a **Etapa 4 do guia da imersão**: com o Claude aberto, cole o pedido de instalação do
GitHub, responda "sim" e espere. No fim ele mostra o endereço do painel e o link de primeiro
acesso. Crie a senha do painel (12+ caracteres) e o primeiro agente.

> No WSL o painel abre no próprio laptop em `https://localhost`. Se o navegador avisar que o
> certificado não é confiável, é esperado nesse caso: clique em "Avançado" → "Continuar".

## Passo 4. Rodar o arquivo de configuração

Ele limita o Ubuntu a 8 GB de RAM (sobra memória para o programa dos guias), liga o que o DG
Claw precisa e faz o Ubuntu abrir sozinho toda vez que você entra no Windows.

1. Copie esta pasta para `C:\servidor` (ou qualquer lugar fácil).
2. Abra o PowerShell **como administrador** (igual ao passo 1).
3. Rode:

   ```
   cd C:\servidor
   ```

   ```
   powershell -ExecutionPolicy Bypass -File .\configurar-servidor.ps1
   ```

4. No fim ele mostra o status do DG Claw. Se aparecer tudo "ativo"/"running", está certo.

## Passo 5. Energia: o laptop não pode dormir

Configurações do Windows (Iniciar → engrenagem):

1. **Sistema → Energia e bateria → Tela, suspensão e hibernação**:
   - "Quando conectado, desligar a tela após": 10 minutos (pode desligar a tela).
   - "Quando conectado, colocar o dispositivo em suspensão após": **Nunca**.
   - "Quando conectado, hibernar após" (se aparecer): **Nunca**.
2. **Fechar a tampa**: Iniciar → digite **Painel de Controle** → Hardware e Sons → Opções de
   Energia → "Escolher a função do fechamento da tampa" → em **Na tomada**, escolha
   **Nada a fazer** → Salvar alterações.
3. **Lenovo Vantage** (app da Lenovo, se tiver): ative o **Modo de conservação da bateria**.
   Ele segura a carga em ~60% e evita estufar a bateria com o laptop sempre na tomada.
4. **Windows Update**: Configurações → Windows Update → Opções avançadas → **Horário ativo**
   → Manualmente → das 6h às 23h. Assim ele só reinicia de madrugada.
5. **Entrar sozinho depois de reiniciar** (opcional, recomendado): sem isso, depois de um
   reinício do Windows Update o agente fica parado até você digitar a senha. Se o laptop fica
   em casa e só você usa, pode deixar o login automático; me peça que eu passo como fazer.

## Passo 6. Acessar de fora de casa (Tailscale)

1. No laptop: baixe em **tailscale.com/download**, instale e entre com sua conta Google.
2. No celular: instale o app **Tailscale** e entre com a **mesma** conta.
3. No app do celular aparece o laptop com um endereço `100.x.x.x`. No navegador do celular,
   abra `https://100.x.x.x` para chegar no painel do DG Claw (de qualquer lugar).

Nada fica aberto na internet: só os seus aparelhos dentro do Tailscale enxergam o laptop.
**Não abra portas no roteador.**

## Passo 7. Trazer a Lúcia (comigo)

Instale também o **Claude** (app desktop) no Windows do laptop e abra uma sessão comigo lá.
Eu recrio a Lúcia: Telegram, agenda e as rotinas das 7h e das 20h. Depois desligamos a Lúcia
do PC do consultório: **as duas não podem usar o mesmo bot ao mesmo tempo**.

---

## Se algo parar

| Sintoma | O que fazer |
|---|---|
| Painel não abre | Abra o Ubuntu e rode `dgclaw status`. Se algo estiver parado, peça ao `claude` dentro do Ubuntu: "o dgclaw parou, verifica e sobe de novo". |
| Parou depois de reiniciar | Entre no Windows com sua senha; o Ubuntu sobe sozinho em ~1 minuto. |
| Laptop lento | Feche o Ubuntu não resolve (ele volta sozinho). Me chame para baixar o limite de 8 GB para 6 GB. |
| Quero desligar o servidor de vez | Iniciar → Agendador de Tarefas → apague "DG Claw - manter Ubuntu ligado"; depois `wsl --shutdown` no PowerShell. |
