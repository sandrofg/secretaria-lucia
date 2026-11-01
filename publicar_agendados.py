"""Publicador dos posts aprovados pelo Sandro.

Roda a cada 10 min (Agendador de Tarefas do Windows) ou na hora, quando o Argos dispara a
tarefa. Se não houver post vencido, sai sem chamar o Claude (custo zero). Se houver, chama
`claude -p` com SÓ as ferramentas do Zapier/Drive liberadas e um roteiro fixo:
compartilhar a foto da pasta Lucia-posts (só ver), publicar, tirar o link público, avisar.

O Argos (que conversa com o Sandro) não torna nada público: ele só põe o post aprovado na
fila. Quem executa é este processo, que não recebe mensagens de ninguém.

Fila: posts-agendados.json — lista de
{id, quando (ISO com fuso), rede: "instagram"|"linkedin", conta_id, conta_nome, legenda,
 imagem_arquivo (nome do arquivo em G:/My Drive/Lucia-posts), status, detalhe}
"""
import json
import os
import re
import shutil
import subprocess
from datetime import datetime, timezone
from pathlib import Path

PASTA = Path(__file__).resolve().parent
FILA = PASTA / "posts-agendados.json"
TRAVA = PASTA / "posts-agendados.lock"
POSTS_DRIVE = Path("G:/My Drive/Lucia-posts")
_CLAUDE_NPM = Path(os.environ.get("APPDATA", "")) / "npm/node_modules/@anthropic-ai/claude-code/bin/claude.exe"
# Instalação via npm (PC antigo) ou nativa (~/.local/bin, Legion): usa a que existir.
CLAUDE = _CLAUDE_NPM if _CLAUDE_NPM.exists() else Path(shutil.which("claude") or _CLAUDE_NPM)
ENVIAR = PASTA / "enviar-telegram.ps1"
FERRAMENTAS = ["mcp__claude_ai_Zapier__execute_zapier_write_action",
               "mcp__claude_ai_Zapier__inspect_zapier_actions",
               "mcp__claude_ai_Google_Drive__search_files"]
SEM_JANELA = 0x08000000  # CREATE_NO_WINDOW: roda sem piscar janela
ESPERA_SYNC_S = 120      # tempo mínimo desde a cópia, para o Drive terminar de subir a foto
NOME_SEGURO = re.compile(r"^[\w.-]+\.(jpe?g|png)$", re.I)

ROTEIRO_INSTAGRAM = """Siga exatamente estes passos, com estes valores, e nada mais:
1. Ache o arquivo de nome exato ARQUIVO na pasta "Lucia-posts" do Google Drive com
   search_files (ex.: title = 'ARQUIVO'). Se não achar, responda ERRO: foto não sincronizada.
   Use só um arquivo cujo pai seja a pasta Lucia-posts.
2. Torne-o visível por link com execute_zapier_write_action: selected_api "GoogleDriveCLIAPI",
   action "share_file", tool_name "google_drive_add_file_sharing_preference",
   params {"file_id": <ID>, "permission": "public_link_view"}. Nunca outra permissão.
3. Publique com execute_zapier_write_action: selected_api "InstagramBusinessCLIAPI",
   action "publish_media_v2", tool_name "instagram_for_business_publish_photo_s",
   params {"instagramPageId": CONTA_ID,
           "media": ["https://drive.google.com/uc?export=download&id=<ID>"],
           "caption": LEGENDA}.
4. Sempre (mesmo se o passo 3 falhar) remova o link público: execute_zapier_write_action
   selected_api "GoogleDriveCLIAPI", action "remove_file_permission",
   params {"file": <ID>, "permission_id": "anyoneWithLink"}.
5. Responda só PUBLICADO ou ERRO: <motivo curto>."""

ROTEIRO_LINKEDIN = """Publique no LinkedIn com execute_zapier_write_action: selected_api
"LinkedInCLIAPI", action "share", params {"comment": LEGENDA, "visibility__code": "anyone"}.
Uma única chamada. Responda só PUBLICADO ou ERRO: <motivo curto>."""


def avisar(texto):
    subprocess.run(["powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", str(ENVIAR),
                    "-Texto", texto], capture_output=True, creationflags=SEM_JANELA)


def publicar(post):
    valores = {"CONTA_ID": post.get("conta_id", ""), "LEGENDA": post["legenda"]}
    if post["rede"] == "instagram":
        arquivo = post.get("imagem_arquivo", "")
        if not NOME_SEGURO.match(arquivo) or not (POSTS_DRIVE / arquivo).is_file():
            return False, "ERRO: foto não encontrada em Lucia-posts"
        valores["ARQUIVO"] = arquivo
        roteiro = ROTEIRO_INSTAGRAM
    else:
        roteiro = ROTEIRO_LINKEDIN
    pedido = (roteiro + "\n\nValores (use sem alterar uma letra):\n"
              + json.dumps(valores, ensure_ascii=False))
    pasta_vazia = PASTA / "publicador"
    pasta_vazia.mkdir(exist_ok=True)
    r = subprocess.run(
        [str(CLAUDE), "-p", "--tools", "", "--allowedTools", *FERRAMENTAS, "--output-format", "json"],
        input=pedido, capture_output=True, text=True, encoding="utf-8", cwd=pasta_vazia,
        timeout=900, creationflags=SEM_JANELA)
    try:
        texto = (json.loads(r.stdout).get("result") or "").strip()
    except ValueError:
        texto = "ERRO: resposta inválida do Claude"
    return texto.upper().startswith("PUBLICADO"), texto[:200]


def pronto(post, agora):
    if post.get("status") != "agendado" or post.get("rede") not in ("instagram", "linkedin"):
        return False
    if datetime.fromisoformat(post["quando"]) > agora:
        return False
    if post["rede"] == "instagram":  # dá tempo ao Drive de subir a foto
        foto = POSTS_DRIVE / post.get("imagem_arquivo", "_")
        if foto.is_file() and agora.timestamp() - foto.stat().st_mtime < ESPERA_SYNC_S:
            return False
    return True


def main():
    if not FILA.exists() or TRAVA.exists():
        return
    fila = json.loads(FILA.read_text(encoding="utf-8"))
    agora = datetime.now(timezone.utc)
    vencidos = [p for p in fila if pronto(p, agora)]
    if not vencidos:
        return
    TRAVA.write_text(agora.isoformat())
    try:
        for p in vencidos:
            ok, detalhe = publicar(p)
            p["status"], p["detalhe"] = ("publicado" if ok else "erro"), detalhe
            p["publicado_em"] = datetime.now().astimezone().isoformat(timespec="minutes")
            FILA.write_text(json.dumps(fila, ensure_ascii=False, indent=1), encoding="utf-8")
            onde = f'{p["rede"]} ({p.get("conta_nome", "")})'
            avisar(f"Publicado no {onde}: \"{p['legenda'][:60]}\"" if ok
                   else f"Não consegui publicar no {onde}. {detalhe}. Ficou na fila como erro; me diga se tento de novo.")
    finally:
        TRAVA.unlink(missing_ok=True)


if __name__ == "__main__":
    main()
