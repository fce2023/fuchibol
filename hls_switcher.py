import os
import time
import shutil

HLS_DIR = "/home/oyon/sehuacho/fuchibol/backend/srs_hls"
LIVE_DIR = os.path.join(HLS_DIR, "live")
OFFLINE_FILE = os.path.join(HLS_DIR, "offline", "offline.m3u8")
MASTER_FILE = os.path.join(LIVE_DIR, "channel_1.m3u8")

def get_active_iptv():
    files = [f for f in os.listdir(LIVE_DIR) if f.startswith("iptv_") and f.endswith(".m3u8")]
    if not files:
        return None
    # Priorizar iptv_1.m3u8 si existe, si no el primero que encuentre
    if "iptv_1.m3u8" in files:
        return "iptv_1.m3u8"
    return files[0]

def update_master():
    obs_file = os.path.join(LIVE_DIR, "channel_1_real.m3u8") # Cambiaremos SRS para que use este nombre
    # Si SRS genera channel_1.m3u8 directamente, tenemos que ser cuidadosos para no entrar en loop
    # Por ahora, buscaremos si existe channel_1.m3u8 pero que NO sea nuestro link
    
    target = None
    
    # 1. Buscar OBS (asumimos que SRS genera algo con channel_1 pero diferente a nuestro master si lo borramos)
    # Para esta lógica, vamos a usar un nombre diferente para el MASTER y que el usuario use ese.
    
    # Intentar detectar OBS activo (buscando fragmentos .ts de channel_1 que sean recientes)
    ts_files = [f for f in os.listdir(LIVE_DIR) if f.startswith("channel_1-") and f.endswith(".ts")]
    if ts_files:
        # Verificar si alguno es reciente (últimos 30 segundos)
        recent = False
        for ts in ts_files:
            if time.time() - os.path.getmtime(os.path.join(LIVE_DIR, ts)) < 30:
                recent = True
                break
        if recent:
            # Aquí hay un truco: SRS genera channel_1.m3u8. Si nosotros lo sobreescribimos,
            # perdemos la fuente. Así que el MASTER para la OLT debería ser otro nombre.
            target = "channel_1.m3u8"

    # 2. Si no hay OBS, buscar IPTV
    if not target:
        target = get_active_iptv()
    
    # 3. Si no hay nada, Offline
    if not target:
        return OFFLINE_FILE
    
    return os.path.join(LIVE_DIR, target)

# Lógica simplificada para la OLT: usaremos "master.m3u8" como el link definitivo
MASTER_OLT = os.path.join(LIVE_DIR, "master.m3u8")

print("HLS Switcher started...")
while True:
    try:
        source = None
        # Buscar OBS
        if os.path.exists(os.path.join(LIVE_DIR, "channel_1.m3u8")) and os.path.getsize(os.path.join(LIVE_DIR, "channel_1.m3u8")) > 0:
            # Verificar que no sea nuestro propio master.m3u8 (no aplica aquí por nombre)
            source = os.path.join(LIVE_DIR, "channel_1.m3u8")
        else:
            # Buscar IPTV
            iptv = get_active_iptv()
            if iptv:
                source = os.path.join(LIVE_DIR, iptv)
            else:
                source = OFFLINE_FILE
        
        if source:
            # En lugar de symlink (que da problemas en HTTP), COPIAMOS el contenido
            # para que Nginx siempre vea un archivo real y no se corte
            with open(source, 'r') as f:
                content = f.read()
            
            # Ajustar rutas internas si es necesario (HLS usa rutas relativas, así que suele funcionar)
            with open(MASTER_OLT, 'w') as f:
                f.write(content)
                
    except Exception as e:
        print(f"Error: {e}")
    
    time.sleep(2)
