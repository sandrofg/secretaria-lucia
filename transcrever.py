r"""Transcreve um audio (voz do Telegram, .oga/.ogg/.mp3/.m4a) para texto em portugues.

Uso: .venv-voz\Scripts\python.exe transcrever.py <arquivo>
Roda local (faster-whisper, modelo "small", CPU). Nada sai do computador.
"""
import sys
from pathlib import Path

import av
import numpy as np
from faster_whisper import WhisperModel

MODELOS = Path(__file__).parent / "modelos-voz"


def carregar_audio(caminho: str) -> np.ndarray:
    """Decodifica para 16 kHz mono float32 (o decodificador do faster-whisper quebra com o PyAV 19)."""
    resampler = av.AudioResampler(format="s16", layout="mono", rate=16000)
    blocos = []
    with av.open(caminho) as arquivo:
        for quadro in arquivo.decode(audio=0):
            for saida in resampler.resample(quadro):
                blocos.append(saida.to_ndarray().reshape(-1))
    for saida in resampler.resample(None):
        blocos.append(saida.to_ndarray().reshape(-1))
    if not blocos:
        return np.zeros(0, dtype=np.float32)
    return np.concatenate(blocos).astype(np.float32) / 32768.0


def main() -> None:
    if len(sys.argv) != 2 or not Path(sys.argv[1]).is_file():
        sys.exit("uso: transcrever.py <arquivo de audio>")
    modelo = WhisperModel("small", device="cpu", compute_type="int8", download_root=str(MODELOS))
    trechos, _ = modelo.transcribe(carregar_audio(sys.argv[1]), language="pt", vad_filter=True, beam_size=5)
    texto = " ".join(t.text.strip() for t in trechos).strip()
    sys.stdout.reconfigure(encoding="utf-8")
    print(texto or "(audio sem fala reconhecivel)")


if __name__ == "__main__":
    main()
