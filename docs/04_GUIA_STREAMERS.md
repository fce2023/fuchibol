# Guía para Streamers (OBS Studio)

La plataforma Fuchibol soporta ingesta de video mediante el protocolo estándar **RTMP**, haciéndola 100% compatible con software profesional de transmisión como OBS Studio o vMix.

## Configuración en OBS Studio

Ve a los ajustes de OBS y dirígete a la pestaña de **Emisión (Stream)**.

1. **Servicio:** Personalizado (Custom...)
2. **Servidor (URL):** `rtmp://fuchibol.elconsejosupremo.com/live`
3. **Clave de Transmisión (Stream Key):** `TU_CLAVE_DE_TRANSMISION`

*La clave de transmisión la puedes encontrar en tu panel de administración. Tiene un formato similar a `live_abc123...`.*

## Configuración de Salida de Video (Recomendado)

Para asegurar la menor latencia posible y evitar cortes por saturación, ve a la pestaña de **Salida (Output)** en OBS y ajusta:

*   **Codificador (Encoder):** H.264 (x264 o NVIDIA NVENC).
*   **Control de Frecuencia (Rate Control):** CBR (Constant Bitrate).
*   **Tasa de Bits (Bitrate):** 
    *   Para 1080p 60fps: `6000 Kbps`
    *   Para 720p 60fps: `4500 Kbps`
    *   Para 720p 30fps: `3000 Kbps`
*   **Intervalo de Fotogramas Clave (Keyframe Interval):** `2 s` (¡MUY IMPORTANTE para WebRTC y HLS de baja latencia!).
*   **Perfil (Profile):** `main` o `high`.

## Consideraciones sobre los Cortes de Señal
En el servidor se han optimizado los *buffers* de Nginx y del servidor SRS para evitar micro-desconexiones. Sin embargo, el streamer debe asegurarse de no emitir a un Bitrate mayor al que soporta su velocidad de **Subida** (Upload speed) de Internet.
