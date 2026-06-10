#!/bin/bash
# Script para desplegar la configuración de Nginx y configurar SSL
set -e

# Asegurar que se ejecuta con sudo
if [ "$EUID" -ne 0 ]; then
  echo "Por favor, ejecuta este script como root o usando sudo: sudo bash $0"
  exit 1
fi

echo "==> Copiando archivo de configuración de Nginx..."
cp /home/oyon/sehuacho/fuchibol/nginx_host_fuchibol.conf /etc/nginx/sites-available/fuchibol.elconsejosupremo.com

echo "==> Creando enlace simbólico para habilitar el sitio..."
ln -sf /etc/nginx/sites-available/fuchibol.elconsejosupremo.com /etc/nginx/sites-enabled/

echo "==> Probando la configuración de Nginx..."
nginx -t

echo "==> Recargando Nginx..."
systemctl reload nginx

echo "==> Reiniciando contenedores Docker para aplicar el cambio de puertos y variables..."
cd /home/oyon/sehuacho/fuchibol
docker compose down
docker compose up -d

echo "==> Ejecutando Alembic para asegurar migraciones..."
docker compose exec -T backend alembic upgrade head || echo "Alerta: No se pudo correr alembic (puede que no haya migraciones pendientes)"

echo "==> Solicitando certificado SSL con Certbot..."
certbot --nginx -d fuchibol.elconsejosupremo.com --non-interactive --agree-tos --email admin@grupotufibra.com --redirect

echo "==> Despliegue completado con éxito!"
