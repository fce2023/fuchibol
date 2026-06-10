# WebSockets y Chat en Vivo

Para garantizar ultra baja latencia y alta concurrencia (capacidad de soportar a miles de usuarios interactuando al mismo tiempo sin caída del servidor), Fuchibol utiliza un **Hub de WebSockets Nativos** escritos en Go. 

Se ha abandonado el uso de librerías con sobrecarga como `Socket.IO`. Todo el chat utiliza el estándar puro de los navegadores.

## Conexión al Hub de Chat

La URL de conexión al chat es:
`wss://fuchibol.elconsejosupremo.com/ws/chat?channel=<ID_DEL_CANAL>`

*Nota: Utilizamos `wss://` porque la plataforma principal opera sobre HTTPS.*

### Ejemplo de cliente en Javascript (Frontend)

En el lado cliente, la conexión al chat es extremadamente sencilla y ligera, sin necesidad de librerías externas (NPM).

```javascript
// 1. Establecer conexión
const channelID = 1;
const chatSocket = new WebSocket(`wss://fuchibol.elconsejosupremo.com/ws/chat?channel=${channelID}`);

// 2. Escuchar mensajes entrantes (Broadcast del servidor)
chatSocket.onmessage = function(event) {
    const message = JSON.parse(event.data);
    console.log("Nuevo mensaje recibido:", message);
    
    // Ejemplo de payload recibido:
    // {
    //    "channel_id": 1,
    //    "content": "¡Hola a todos!",
    //    "user_id": 45,
    //    "username": "fan_del_fuchibol"
    // }
};

// 3. Enviar un mensaje (Al presionar "Enviar" en la caja de texto)
function sendMessage(text, userId, username) {
    const payload = {
        channel_id: channelID,
        content: text,
        user_id: userId,
        username: username
    };
    chatSocket.send(JSON.stringify(payload));
}

// 4. Manejo de desconexiones
chatSocket.onclose = function(event) {
    console.log("Desconectado del chat, reintentando...");
    // Aquí puedes implementar una lógica de auto-reconexión usando setTimeout
};
```

## Arquitectura del Hub en el Servidor

Internamente, el backend en Go funciona de la siguiente manera:
1.  **Register:** Cuando un WebSocket se conecta, el Hub lo enlista en un mapa de conexiones activas asignadas a la "Sala" (`channelID`).
2.  **Broadcast:** Cuando alguien envía un mensaje, este se manda al canal interno de Broadcast.
3.  **Distribución:** Go utiliza una Goroutine para reenviar de forma paralela ese mensaje a todos los clientes que estén conectados a la misma sala, asegurando un retraso cercano a los `0 milisegundos`.
4.  **Persistencia:** Simultáneamente, el servidor guarda el mensaje en Postgres para que quede guardado en el historial, **sin bloquear** el envío en vivo.
