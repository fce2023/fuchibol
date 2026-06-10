# Despliegue y Operaciones

Todo el ecosistema de Fuchibol se despliega usando **Docker Compose**, lo que facilita el levantamiento de toda la infraestructura con un solo comando.

## 1. Comandos Principales

Para iniciar la plataforma completa (construyendo las imágenes de Go si hubo cambios):
```bash
docker compose up -d --build
```

Para detener todos los servicios de forma segura:
```bash
docker compose down
```

Para ver el estado de los contenedores (Saber si están "Up" o si están reiniciándose por error):
```bash
docker compose ps
```

## 2. Lectura de Logs en Producción

Monitorear los logs es fundamental para detectar problemas en tiempo real. 

Ver todos los logs simultáneamente (No recomendado por exceso de ruido):
```bash
docker compose logs -f
```

### Ver Logs Específicos

**Ver logs de la API y WebSockets (Backend Go):**
Aquí podrás ver cada petición HTTP y la entrada/salida de usuarios al chat.
```bash
docker compose logs -f backend
```

**Ver logs de tareas en segundo plano (Worker Asynq):**
Aquí observarás si el proceso de reenviar el stream a Twitch/YouTube está fallando o consumiendo mucha CPU.
```bash
docker compose logs -f celery
```

**Ver logs del Streaming de Video (SRS):**
Útil si los usuarios reportan que "no se ve el video" o para ver errores de conexión RTMP.
```bash
docker compose logs -f srs
```

**Ver logs del Proxy Reverso (Nginx):**
Ideal para auditar quién está accediendo, direcciones IP y errores 502/504 de red.
```bash
docker compose logs -f nginx
```

## 3. Actualización de Código
Si haces un cambio en el código fuente de Go (directorio `backend-go`), solo necesitas hacer:
```bash
docker compose up -d --build backend celery
```
Esto reconstruirá el binario ultra rápido de Go y reiniciará únicamente esos dos contenedores sin tumbar la base de datos ni el servidor de video.
