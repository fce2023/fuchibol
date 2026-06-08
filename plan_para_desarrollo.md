Plan de Arquitectura Revisado — Plataforma de Streaming Web
Principios rectores

Flujos de error tan diseñados como flujos exitosos. Cada componente tiene un estado de falla explícito.
CDN-ready, no CDN-dependent. La infraestructura se diseña para que agregar Cloudflare o cualquier CDN sea configuración, no refactoring.
Moderación y seguridad desde el modelo de datos, no como capa posterior.
Observabilidad operacional suficiente para detectar problemas antes que los usuarios.


1. Stack tecnológico y Dockerización
El stack original se mantiene íntegro, con las siguientes adiciones y requisitos de infraestructura:

- **Celery + Redis como broker de tareas:** formalizado como componente de primera clase (no solo mencionado como "un worker"). Gestiona transcodificación, bulk inserts de chat, y notificaciones.
- **Prometheus + Grafana:** (opcional Fase 2, pero los endpoints `/metrics` se exponen desde Fase 1 para no tener que refactorizar después).
- **Despliegue 100% Dockerizado:** Toda la plataforma debe estar completamente contenida en Docker. Para mantener la consistencia en el ecosistema, todos los contenedores deben nombrarse explícitamente utilizando el prefijo `fuchibol_`. Específicamente:
  - `fuchibol_frontend`: Contenedor para la aplicación cliente.
  - `fuchibol_backend`: Contenedor para la API de FastAPI.
  - `fuchibol_db`: Contenedor para la base de datos PostgreSQL.
  - `fuchibol_redis`: Contenedor para el broker de Redis.
  - `fuchibol_srs`: Contenedor para el Simple Realtime Server (SRS).
  - `fuchibol_nginx`: Contenedor para Nginx (servidor HLS y proxy inverso).
  - `fuchibol_celery`: Contenedor para los workers de procesamiento.


2. Seguridad ampliada
Stream keys:

Generadas con secrets.token_urlsafe(32), almacenadas como hash en PostgreSQL (nunca en texto plano).
Campo key_expires_at en Channels: TTL configurable (default 90 días). Un endpoint autenticado permite rotación sin perder el canal ni el historial.
Rate limiting en el webhook /on_publish: máximo 10 intentos por IP por minuto, implementado con Redis + sliding window.

Acceso a VODs:

Los archivos en S3/MinIO son privados por defecto (bucket policy denegación pública).
El backend genera URLs pre-firmadas con TTL de 4 horas al momento de la solicitud.
Un campo is_public en Recordings permite que el streamer decida si el VOD es accesible sin autenticación (genera URL temporal igualmente, no expone el bucket).

Autorización por roles:
Los roles definidos (admin, mod, streamer, user) tienen políticas explícitas documentadas antes de implementar el primer endpoint:
AcciónuserstreamermodadminVer streams✓✓✓✓Iniciar stream—✓✓✓Borrar mensajes de chat—solo su canal✓✓Banear usuarios——✓✓Acceder a /admin/*———✓

3. Arquitectura de datos revisada
Modelos SQLAlchemy con campos completos:
Users: id, email (unique, ci collation), hashed_password, username (unique), avatar_url, role, is_banned, created_at.
Channels: id, user_id (FK), name, description, category, stream_key_hash, key_expires_at, is_live, slowmode_seconds (default 0), updated_at.
Streams: id, channel_id (FK), title, start_time, end_time, status (live | ended | interrupted), peak_viewers, created_at.
Recordings: id, stream_id (FK), s3_url, duration_seconds, status (pending | processing | ready | failed), is_public, created_at.
ChatMessages: id, channel_id (FK), user_id (FK), content, timestamp, is_deleted, deleted_by (FK nullable), deletion_reason.
Followers: follower_id (FK), channel_id (FK), created_at. Clave primaria compuesta.
BlockedTerms: id, channel_id (FK nullable — null = global), term, created_by (FK), created_at.
Índices definidos desde el diseño (no como optimización posterior):
ChatMessages: (channel_id, timestamp DESC)
Channels:     (is_live, category), (user_id)
Streams:      (channel_id, status, start_time DESC)
Followers:    (channel_id), (follower_id)
Users:        (email) UNIQUE, (username) UNIQUE
Métricas de viewers concurrentes:
No se almacena solo peak_viewers. Redis mantiene un contador activo con INCR/DECR por canal. Un worker toma snapshots cada 60 segundos a una tabla ViewerSnapshots (stream_id, timestamp, count) para generar la curva de audiencia del stream.

4. Flujos de datos críticos (con manejo de errores)
A. Ingesta RTMP y autenticación:
OBS → SRS (RTMP) → POST /api/v1/streams/webhook/on_publish
                  ↓
         FastAPI valida:
         1. stream_key_hash en PostgreSQL
         2. key_expires_at > now()
         3. user.is_banned == False
         4. Rate limit por IP (Redis)
                  ↓
    403 → SRS corta conexión    200 → SRS acepta stream
                                      ↓
                              FastAPI: channel.is_live = True
                                       Celery: notifica seguidores
                                       Redis: inicia contador viewers
B. Chat de alta concurrencia:
Cliente → Socket.IO → FastAPI
                      ↓
            1. Validar autenticación JWT
            2. Verificar slowmode (Redis: last_message_timestamp por user+channel)
            3. Filtrar BlockedTerms (caché en Redis, TTL 5 min)
            4. Publicar a Redis Pub/Sub: channel:{id}:chat
                      ↓
         Todos los nodos WS suscritos distribuyen a clientes
                      ↓
         Worker Celery: bulk insert cada 5s a PostgreSQL
El slowmode se valida en el servidor, nunca solo en el cliente. Si now() - last_message < slowmode_seconds, se devuelve un evento de error con el tiempo restante.
C. Finalización del stream y generación de VOD:
OBS desconecta → SRS → POST /api/v1/streams/webhook/on_unpublish
                        ↓
               FastAPI: channel.is_live = False
                        stream.status = 'ended'
                        stream.end_time = now()
                        Encola tarea Celery: process_vod(stream_id)
                        ↓
               Worker FFmpeg (Celery):
               try:
                 recording.status = 'processing'
                 FFmpeg empaqueta a MP4 (con NVENC si disponible)
                 Sube a S3/MinIO
                 recording.s3_url = url
                 recording.status = 'ready'
                 Notifica al streamer via WebSocket
               except:
                 recording.status = 'failed'
                 Encola en dead-letter queue
                 Alerta operacional
               finally:
                 Purga archivos locales garantizado (independiente del resultado)
Reintentos: max_retries=3, backoff exponencial (60s, 300s, 900s). Después del tercer fallo, la tarea va a dead-letter y requiere intervención manual.
D. Detección de stream caído:
Un worker periódico (cada 30 segundos) verifica todos los canales con is_live = True. Si SRS no reporta actividad para ese canal en los últimos 45 segundos, el sistema:

Marca channel.is_live = False y stream.status = 'interrupted'.
Envía evento WebSocket a los viewers del canal con estado stream_interrupted.
El cliente muestra pantalla de "stream interrumpido" con opción de reconectar.


5. Preparación CDN (sin implementar, sin deuda técnica)
La infraestructura se diseña para que agregar un CDN sea cero cambios en el código, solo configuración de red.
En Nginx: Todos los segmentos HLS se sirven con headers ya configurados:
Cache-Control: public, max-age=6        # para .m3u8 (duración del segmento)
Cache-Control: public, max-age=31536000 # para .ts (inmutable una vez generado)
X-Content-Type-Options: nosniff
Vary: Accept-Encoding
CORS: Headers configurados en Nginx para todos los paths de HLS, no en la aplicación. Permite que cualquier CDN o dominio de reproducción funcione sin tocar el backend.
URLs absolutas: El reproductor HLS siempre recibe URLs absolutas del backend (no relativas). Cuando se agregue un CDN, solo se cambia la variable de entorno HLS_BASE_URL; el reproductor no cambia.
Health check endpoint: GET /health/hls/{channel_id} devuelve el estado del stream y la URL del .m3u8. Un CDN puede usar este endpoint para validar el origen antes de cachear. En Fase 1, lo consume el frontend directamente.

6. Observabilidad desde el día uno
Endpoint /admin/health (solo role=admin): devuelve JSON con estado de SRS, workers Celery activos, profundidad de cola, conexiones WebSocket activas, y canales live actuales.
Endpoint /metrics: expone métricas en formato compatible con Prometheus (sin necesitar instalar Prometheus en Fase 1; el endpoint existe para cuando se necesite).
Structured logging: Todos los logs en formato JSON con campos: timestamp, level, service, stream_id (cuando aplica), user_id (cuando aplica), duration_ms. Esto permite filtrar en cualquier agregador de logs sin cambiar el código.
Alertas operacionales Fase 1 (implementadas como logs estructurados con nivel CRITICAL, fáciles de enganchar a cualquier sistema de alertas después):

Stream marcado como interrupted por el watchdog.
VOD en estado failed después de todos los reintentos.
Cola Celery con más de 50 tareas pendientes.
Webhook /on_publish rechazado 10+ veces desde la misma IP en 1 minuto.


7. Entregables Fase 1 revisados
Los entregables técnicos originales se mantienen, con estos agregados de producto:
EntregableDescripcióndocker-compose.ymlFastAPI, PostgreSQL, Redis, SRS, Nginx, Celery workersrs.confHLS + webhooks + headers de caché predispuestosAuth endpointsJWT + OAuth Google + rotación de stream keyWebhook /on_publish + /on_unpublishCon rate limiting y encolado de VODWorker CeleryTranscodificación VOD + bulk insert chat + notificacionesComponente reproductor Vue 3hls.js + reconexión automática + pantalla de stream terminadoComponente chat Vue 3Socket.IO + slowmode UI + moderación básicaPágina de canalEstado online/offline en tiempo real + título + contador viewersWatchdog de streamsWorker periódico que detecta streams caídos/admin/healthPanel de salud del sistema

8. Orden de implementación recomendado
El plan original no define el orden, lo que genera dependencias implícitas. El orden correcto para que cada entregable sea testeable de forma independiente:

Modelos de datos + migraciones Alembic — la base de todo lo demás.
Docker Compose con todos los servicios levantando correctamente.
Autenticación (JWT + OAuth) — necesaria antes de cualquier endpoint protegido.
SRS + webhook de ingesta — verificar que el flujo RTMP → validación → HLS funciona end-to-end.
Reproductor HLS — validar con un stream real antes de construir el chat.
Chat con Redis Pub/Sub — construir sobre un stream ya funcional.
Worker Celery (VOD + bulk chat insert + watchdog).
Página de canal + notificaciones — superficie de producto que integra todo lo anterior.
Moderación (slowmode, borrado de mensajes, ban).
Observabilidad (/health, /metrics, structured logging).