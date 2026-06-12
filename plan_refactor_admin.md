# Plan de Refactorización: Panel de Administración Fuchibol

Este documento detalla el plan para modernizar el panel de administración, agregando navegación lateral, soporte móvil, gestión avanzada de IPTV y un dashboard de analíticas.

## 1. Rediseño de Interfaz (UI/UX)

### Objetivo
Pasar de una vista de desplazamiento único a una experiencia de "App" con navegación lateral (Sidebar).

### Tareas
- [ ] **Sidebar de Navegación**: Crear un componente lateral con los siguientes accesos:
  - 📊 Dashboard (Resumen y Estadísticas)
  - 👤 Perfil (Datos de usuario y canal)
  - 📡 Transmisión (Claves OBS y Gestor IPTV)
  - 🗓️ Agenda (Gestión de eventos)
  - 👥 Seguidores (Lista de comunidad)
- [ ] **Adaptabilidad Móvil**:
  - Implementar un menú hamburguesa o barra inferior para pantallas < 768px.
  - Asegurar que todas las tarjetas (cards) se ajusten correctamente al ancho de pantalla.
- [ ] **Layout Principal**: Refactorizar `AdminView.vue` para usar un contenedor con `display: grid` o `flex` que separe el Sidebar del Contenido.

## 2. Gestión de IPTV Avanzada

### Objetivo
Permitir asignar un nombre descriptivo a cada enlace de IPTV guardado.

### Tareas
- [ ] **Cambio en Estructura de Datos**: Modificar `iptv_urls` para que sea una lista de objetos: `[{ "url": "...", "name": "..." }]`.
- [ ] **Interfaz de Usuario**:
  - Agregar campo "Nombre de TV/Canal" en el formulario de adición.
  - Mostrar el nombre en la lista de enlaces, con la URL en segundo plano.
- [ ] **Migración Silenciosa**: Asegurar que los enlaces antiguos (solo strings) sigan funcionando o se conviertan al nuevo formato al cargar.

## 3. Dashboard de Analíticas (Vistas y Tráfico)

### Objetivo
Visualizar el tráfico del canal, procedencia geográfica y crecimiento de seguidores.

### Tareas
- [ ] **Backend: Registro de Vistas**:
  - Crear tabla `analytics_logs` con: `channel_id`, `ip_address`, `country`, `city`, `user_agent`, `timestamp`.
  - Crear endpoint `POST /api/v1/analytics/track` (Go Backend recomendado).
- [ ] **GeoIP**: Integrar resolución de IP a país/ciudad (usando servicio externo o base de datos local).
- [ ] **Frontend: Tracking**:
  - Llamar al endpoint de tracking desde `VideoPlayer.vue` cuando inicie la reproducción.
- [ ] **Panel de Dashboard**:
  - Gráfico de vistas en el tiempo.
  - Lista de "Top Países".
  - Contador de seguidores totales y nuevos seguidores (últimos 7 días).

## 4. Cronograma de Implementación

### Fase 1: Backend y Analíticas (Día 1)
- Implementación de modelos y endpoints de tracking.
- Integración de GeoIP.

### Fase 2: Lógica de IPTV y Tracking Frontend (Día 2)
- Refactorización de la lógica de guardado de IPTV en `AdminView.vue`.
- Integración de tracking en `VideoPlayer.vue`.

### Fase 3: Rediseño UI y Sidebar (Día 3)
- Creación de componentes de navegación.
- Implementación del nuevo layout responsivo.
- Creación de la pestaña "Dashboard" con gráficos.

---
*Nota: Se priorizará el uso del backend en Go para el tracking de analíticas por su mayor rendimiento en operaciones de alta concurrencia.*
