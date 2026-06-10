# Documentación API REST

La API se encuentra estructurada bajo el estándar RESTful, accesible bajo el prefijo `/api/v1/`.

## 1. Autenticación (`/auth`)

### POST `/api/v1/auth/register`
Crea una nueva cuenta de usuario.

**Body (JSON):**
```json
{
  "username": "nuevo_usuario",
  "email": "correo@ejemplo.com",
  "password": "contraseñasegura"
}
```

### POST `/api/v1/auth/login`
Inicia sesión devolviendo un Token JWT válido.

**Body (JSON):**
```json
{
  "email": "correo@ejemplo.com",
  "password": "contraseñasegura"
}
```
**Respuesta (200 OK):**
```json
{
  "token": "eyJhbGciOiJIUz...",
  "user": { "id": 1, "username": "nuevo_usuario" }
}
```

## 2. Canales (`/channels`)

### GET `/api/v1/channels/`
Lista todos los canales que están actualmente transmitiendo en vivo (`is_live = true`).

### GET `/api/v1/channels/:name`
Obtiene la información pública de un canal en específico.

### PATCH `/api/v1/channels/me`
Permite a un streamer actualizar la información (Título, Categoría) de su transmisión. *Requiere Header `Authorization: Bearer <token>`*.

**Body (JSON):**
```json
{
  "title": "Jugando la final del torneo",
  "category": "Deportes"
}
```

## 3. Webhooks de Servidor (`/webhooks/srs`)
Endpoints internos (No accesibles desde el frontend) usados por el servidor de video SRS para notificar eventos.

*   `POST /webhooks/srs/on_publish`: Llamado cuando alguien inicia streaming en OBS.
*   `POST /webhooks/srs/on_unpublish`: Llamado cuando alguien apaga su OBS.

## 4. Administración (`/admin`)
Rutas restringidas para moderadores.

*   `GET /api/v1/admin/users`: Lista todos los usuarios.
*   `POST /api/v1/admin/users/:id/ban`: Banea a un usuario de la plataforma.
