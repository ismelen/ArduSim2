#!/bin/bash

# Este script construye las imágenes de ArduSim2 localmente, las empaqueta en un archivo .tar
# y las envía a uno o varios nodos a través de SSH para cargarlas en su demonio de Docker.

set -e

SSH_KEY=""

# Parsear opciones
while [[ "$1" == -* ]]; do
    case "$1" in
        -i) 
            SSH_KEY="$2"
            shift 2 
            ;;
        *) 
            echo "Opción desconocida: $1"
            exit 1 
            ;;
    esac
done

if [ -z "$1" ]; then
    echo "Error: Debes especificar al menos un nodo de destino."
    echo "Uso: $0 [-i ruta/a/clave_privada] <usuario@ip-del-nodo> [usuario@ip-nodo-2] ..."
    echo "Ejemplo: $0 -i ./mi_clave.pem ubuntu@192.168.1.100 isma@192.168.1.101"
    exit 1
fi

SSH_OPTS=""
SCP_OPTS=""
if [ -n "$SSH_KEY" ]; then
    SSH_OPTS="-i $SSH_KEY"
    SCP_OPTS="-i $SSH_KEY"
    echo "Usando clave privada SSH: $SSH_KEY"
fi

echo "ArduSim2 - Construcción y Empaquetado de Imágenes"

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/src/.." && pwd)"
cd "$PROJECT_ROOT"

# Selección del tipo de SITL
echo ""
echo "¿Qué Dockerfile deseas usar para la imagen SITL del UAV?"
echo "  1) SITL          (estándar)"
echo "  2) SITL.ADAPTIVE (adaptativo) (13')"
echo ""
read -rp "Introduce tu elección [1/2] (por defecto: 1): " SITL_CHOICE

case "$SITL_CHOICE" in
    2)
        SITL_DOCKERFILE="uav_controller/ardupilot4_5_3/SITL.ADAPTIVE"
        echo "→ Usando Dockerfile: SITL.ADAPTIVE"
        ;;
    *)
        SITL_DOCKERFILE="uav_controller/ardupilot4_5_3/SITL"
        echo "→ Usando Dockerfile: SITL (estándar)"
        ;;
esac
echo ""

echo "Construyendo imágenes desde el código local en: $PROJECT_ROOT"

docker build -t netsim_gateway -f netsim_gateway/Dockerfile netsim_gateway/
docker build -t netsim -f netsim/Dockerfile netsim/
docker build -t logger -f logger/Dockerfile logger/
docker build -t communication_module -f communication_module/Dockerfile communication_module/
docker build -t application -f application/Dockerfile application/
docker build -t copter453 -f "$SITL_DOCKERFILE" uav_controller/ardupilot4_5_3/
docker build -t external_comms -f external_comms/Dockerfile external_comms/

CORE_IMAGES="netsim_gateway netsim logger communication_module application copter453 external_comms"

ALGORITHM_IMAGES=""
echo "Buscando algoritmos dinámicamente en /algorithms..."
for dir in algorithms/*/; do
    if [ -f "${dir}Dockerfile" ]; then
        algo_name=$(basename "$dir")
        echo "-> Construyendo algoritmo: $algo_name"
        docker build -t "$algo_name" -f "${dir}Dockerfile" "$dir"
        ALGORITHM_IMAGES="$ALGORITHM_IMAGES $algo_name"
    fi
done

# 2. Empaquetar las imágenes en un archivo .tar
TAR_FILE="/tmp/ardusim2_images.tar"
echo "Empaquetando las imágenes en $TAR_FILE..."
echo "Este proceso puede tardar unos minutos."

docker save -o "$TAR_FILE" $CORE_IMAGES $ALGORITHM_IMAGES

echo "Empaquetado completado. Tamaño del archivo:"
du -h "$TAR_FILE"

for NODE in "$@"; do
    echo "Procesando nodo: $NODE"
    
    echo "Verificando el estado de Docker en $NODE..."
    ssh $SSH_OPTS "$NODE" '
        if ! command -v docker &> /dev/null; then
            echo "ERROR: Docker no está instalado en este nodo."
            exit 1
        fi
        if ! docker info &> /dev/null; then
            echo "Intentando iniciar Docker..."
            if command -v systemctl &> /dev/null; then
                sudo systemctl start docker || sudo systemctl start docker.service
            else
                sudo service docker start
            fi
            sleep 3
            if ! docker info &> /dev/null; then
                echo "ERROR: No se pudo iniciar Docker en el nodo."
                exit 1
            fi
        fi
    ' || { echo "Fallo al verificar Docker en $NODE. Saltando nodo..."; continue; }

    echo "Enviando archivo tar por SCP a $NODE..."
    scp $SCP_OPTS "$TAR_FILE" "$NODE:/tmp/ardusim2_images.tar"
    
    echo "Cargando imágenes en el Docker de $NODE..."
    ssh $SSH_OPTS "$NODE" "docker load -i /tmp/ardusim2_images.tar && rm /tmp/ardusim2_images.tar"
    
    echo "Imágenes cargadas exitosamente en $NODE"
done

echo "Limpiando archivo local temporal..."
rm "$TAR_FILE"

echo "¡Terminado! Las imágenes se han distribuido y cargado."
