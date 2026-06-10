# Arquitectura de la Plataforma (Fuchibol)

Este documento describe la arquitectura técnica de la plataforma, que combina streaming de video en vivo de ultra baja latencia con un chat interactivo.

## Diagrama de Red y Flujo de Datos

```mermaid
graph TD
    %% Entidades Externas
    Streamer[Streamer (OBS Studio)]
    Viewer[Espectador (Navegador)]
    
    %% Host Físico
    subgraph Host [Servidor Host]
        NginxHost[Nginx Host / Certbot]
        
        %% Red Docker Interna
        subgraph Docker [Contenedores Docker]
            NginxDocker[Nginx Proxy Reverso]
            BackendGo[API Backend & WebSockets (Go)]
            Celery[Asynq Worker (Go)]
            SRS[SRS - Simple Realtime Server]
            Postgres[(Base de Datos Postgres)]
            Redis[(Redis Cache/Queue)]
            Vue[Frontend Vue.js]
        end
    end
    
    %% Conexiones Streamer
    Streamer -- "1. Sube video (RTMP)" --> SRS
    SRS -- "2. Webhook HTTP /on_publish" --> BackendGo
    BackendGo -- "3. Guarda Estado" --> Postgres
    
    %% Conexiones Viewer Video
    Viewer -- "4. Pide video (HTTPS)" --> NginxHost
    NginxHost -- "Proxy_pass" --> NginxDocker
    NginxDocker -- "Pide .m3u8/.ts" --> SRS
    
    %% Conexiones Viewer Web/API
    Viewer -- "Navega Web / API (HTTPS)" --> NginxHost
    NginxHost -- "Proxy_pass" --> NginxDocker
    NginxDocker -- "/api/" --> BackendGo
    NginxDocker -- "/" --> Vue
    
    %% Conexiones Chat (WebSockets)
    Viewer -- "Conecta Chat (WSS)" --> NginxHost
    NginxHost -- "Upgrade: websocket" --> NginxDocker
    NginxDocker -- "WS" --> BackendGo
    BackendGo -- "Guarda Mensaje" --> Postgres
    
    %% Tareas de Fondo
    BackendGo -- "Encola Tarea" --> Redis
    Redis -- "Procesa" --> Celery
```

## Componentes Principales

1. **Nginx Host (Punto de Entrada Seguro)**
   * Se encarga de terminar la conexión SSL (HTTPS) usando certificados de Let's Encrypt o similares para el dominio `fuchibol.elconsejosupremo.com`.
   * Todo el tráfico descifrado se reenvía al ecosistema Docker en el puerto interno 80.

2. **Backend (Go - Fiber)**
   * El núcleo de la plataforma, reescrito desde Python para un rendimiento extremo.
   * Provee las APIs RESTful para autenticación, canales y administración.
   * Maneja el **Hub de WebSockets**, permitiendo a miles de espectadores conectarse al chat usando apenas megabytes de memoria RAM.

3. **SRS (Simple Realtime Server)**
   * Maneja la ingesta de video RTMP enviada desde OBS.
   * Transforma el video en formato HLS (.m3u8/.ts) y WebRTC para ser consumido por el reproductor del cliente.
   * Se comunica con el Backend en Go a través de "Webhooks" (Avisando cuando un streamer se conecta o desconecta).

4. **Base de Datos (Postgres & Redis)**
   * **Postgres:** Persistencia a largo plazo (usuarios, historiales de chat, perfiles).
   * **Redis:** Almacenamiento rápido en caché y gestor de colas para el Worker.

5. **Worker de Fondo (Asynq)**
   * Reemplazo de Celery en Go. Consume tareas pesadas (ej. Restreaming usando FFmpeg hacia YouTube/Twitch) sin afectar el tiempo de respuesta del Backend web.
