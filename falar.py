r"""Transforma texto em mensagem de voz (OGG/Opus, formato de voz do Telegram), com o Piper.

Uso: .venv-voz\Scripts\python.exe falar.py
Lê o texto de voz-msg.txt e grava voz-resposta.ogg, os dois nesta pasta (caminhos fixos,
para a permissão do comando ser sempre a mesma). Roda local: nada sai do computador.

Voz: modelos-voz/piper/<VOZ>.onnx. Para baixar (uma vez):
  .venv-voz\Scripts\python.exe -m piper.download_voices pt_BR-faber-medium --download-dir modelos-voz\piper
"""
import sys
from pathlib import Path

import av
import numpy as np
from piper import PiperVoice

PASTA = Path(__file__).resolve().parent
VOZ = "pt_BR-faber-medium"
MODELO = PASTA / "modelos-voz" / "piper" / f"{VOZ}.onnx"
ENTRADA = PASTA / "voz-msg.txt"
SAIDA = PASTA / "voz-resposta.ogg"
LIMITE_CARACTERES = 1500  # ~1 min e meio de fala; resposta maior vai em texto


def para_ogg(audio: np.ndarray, taxa: int, destino: Path) -> None:
    """Codifica áudio mono int16 em OGG/Opus 48 kHz (o que o Telegram toca como mensagem de voz)."""
    with av.open(str(destino), "w", format="ogg") as saida:
        fluxo = saida.add_stream("libopus", rate=48000, layout="mono")
        fluxo.bit_rate = 32000
        conversor = av.AudioResampler(format="s16", layout="mono", rate=48000)
        quadro = av.AudioFrame.from_ndarray(audio.reshape(1, -1), format="s16", layout="mono")
        quadro.sample_rate = taxa
        for convertido in conversor.resample(quadro) + conversor.resample(None):
            for pacote in fluxo.encode(convertido):
                saida.mux(pacote)
        for pacote in fluxo.encode(None):
            saida.mux(pacote)


def main() -> None:
    if not MODELO.is_file():
        sys.exit(f"voz não encontrada: {MODELO}. Baixe com o comando do topo de falar.py.")
    if not ENTRADA.is_file():
        sys.exit(f"faltou o texto em {ENTRADA.name}")
    texto = ENTRADA.read_text(encoding="utf-8").strip()
    if not texto:
        sys.exit(f"{ENTRADA.name} está vazio")
    if len(texto) > LIMITE_CARACTERES:
        sys.exit(f"texto com {len(texto)} caracteres; o limite para voz é {LIMITE_CARACTERES}. Responda em texto.")

    voz = PiperVoice.load(MODELO)
    trechos = list(voz.synthesize(texto))
    if not trechos:
        sys.exit("o Piper não gerou áudio")
    audio = np.concatenate([t.audio_int16_array for t in trechos]).astype(np.int16)
    para_ogg(audio, trechos[0].sample_rate, SAIDA)
    sys.stdout.reconfigure(encoding="utf-8")
    print(SAIDA)


if __name__ == "__main__":
    main()
