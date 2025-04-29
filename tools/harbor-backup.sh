#!/bin/bash
# Copyright Project Harbor Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

set -e

# Crear directorios necesarios para backup
delete_and_create_dir(){
    rm -rf harbor
    mkdir -p harbor/db harbor/secret
    chmod 777 harbor/db harbor/secret
}

# Lanzar contenedor temporal para respaldar la base de datos
launch_db() {
    if [ -n "$($DOCKER_CMD ps -q)" ]; then
        echo "There is running container, please stop and remove it before backup"
        exit 1
    fi
    $DOCKER_CMD run -d --name harbor-db-backup -v ${PWD}:/backup -v ${harbor_db_path}:/var/lib/postgresql/data ${harbor_db_image} "postgres"
}

# Limpiar contenedor temporal
clean_db() {
    $DOCKER_CMD stop harbor-db-backup
    $DOCKER_CMD rm harbor-db-backup
}

# Esperar que la base de datos esté lista para aceptar conexiones
wait_for_db_ready() {
    set +e
    TIMEOUT=12
    while [ $TIMEOUT -gt 0 ]; do
        $DOCKER_CMD exec harbor-db-backup pg_isready | grep "accepting connections" && break
        TIMEOUT=$((TIMEOUT - 1))
        sleep 5
    done
    if [ $TIMEOUT -eq 0 ]; then
        echo "Harbor DB cannot reach within one minute."
        clean_db
        exit 1
    fi
    set -e
}

# Realizar el volcado de las bases de datos
backup_database() {
    $DOCKER_CMD exec harbor-db-backup sh -c 'pg_dump -U postgres registry > /backup/harbor/db/registry.back'
    $DOCKER_CMD exec harbor-db-backup sh -c 'pg_dump -U postgres postgres > /backup/harbor/db/postgres.back'
    $DOCKER_CMD exec harbor-db-backup sh -c 'pg_dump -U postgres notarysigner > /backup/harbor/db/notarysigner.back'
    $DOCKER_CMD exec harbor-db-backup sh -c 'pg_dump -U postgres notaryserver > /backup/harbor/db/notaryserver.back'
}

# Respaldar archivos de imágenes (registry)
backup_registry() {
    [ -d /data/registry ] && cp -rf /data/registry harbor/ &
    registry_pid=$!
}

# Respaldar archivos de chartmuseum
backup_chart_museum() {
    [ -d /data/chart_storage ] && cp -rf /data/chart_storage harbor/ &
    chartmuseum_pid=$!
}

# Respaldar archivos de Redis
backup_redis() {
    [ -d /data/redis ] && cp -rf /data/redis harbor/
}

# Respaldar llaves y secretos
backup_secret() {
    [ -f /data/secretkey ] && cp /data/secretkey harbor/secret/
    [ -f /data/defaultalias ] && cp /data/defaultalias harbor/secret/
    [ -d /data/secret/keys/ ] && cp -r /data/secret/keys/ harbor/secret/
}

# Crear el archivo comprimido del respaldo
create_tarball() {
    timestamp=$(date +"%Y-%m-%d-%H-%M-%S")
    backup_filename=harbor-$timestamp.tgz

    # Verificar espacio disponible
    ESPACIO_DISPONIBLE=$(df --output=avail . | tail -1)
    TAMANO_DIR=$(du -sk harbor | awk '{print $1}')

    if [[ $ESPACIO_DISPONIBLE -lt $TAMANO_DIR ]]; then
        echo "Error: No hay suficiente espacio en disco." | tee /tmp/backup_error.log
        echo -e "Subject: Error en Backup de Harbor\n\n$(cat /tmp/backup_error.log)" | sendmail sistemas@imm.gub.uy
        return 1
    fi

    # Crear backup usando compresión paralela
    if tar cvf - harbor | pigz > "$backup_filename"; then
        rm -rf harbor
    else
        echo "Error: La compresión falló." | tee /tmp/backup_error.log
        echo -e "Subject: Error en Backup de Harbor\n\n$(cat /tmp/backup_error.log)" | sendmail sistemas@imm.gub.uy
        return 1
    fi

    # Validar integridad del archivo generado
    if ! tar -tzf "$backup_filename" &>/dev/null; then
        echo "Advertencia: El archivo de backup podría estar corrupto." | tee /tmp/backup_error.log
        echo -e "Subject: Advertencia en Backup de Harbor\n\n$(cat /tmp/backup_error.log)" | sendmail sistemas@imm.gub.uy
        return 1
    fi
}

# Opciones del script
usage=$'Uso:
  harbor-backup.sh [opciones]
Opciones:
  --istile   Respaldar en entorno tile
  --dbonly   Respaldar solo bases de datos'

dbonly=false
istile=false

while [ $# -gt 0 ]; do
    case $1 in
        --help)
            echo "$usage"
            exit 0;;
        --dbonly)
            dbonly=true;;
        --istile)
            istile=true;;
        *)
            echo "$usage"
            exit 1;;
    esac
    shift || true
done

# Definir comando Docker según entorno
if [ $istile = true ]; then
    DOCKER_CMD="/var/vcap/packages/docker/bin/docker -H unix:///var/vcap/sys/run/docker/dockerd.sock"
else
    DOCKER_CMD=docker
fi

harbor_db_image=$($DOCKER_CMD images goharbor/harbor-db --format "{{.Repository}}:{{.Tag}}" | head -1)
harbor_db_path="/data/database"

# Detener servicios de Harbor
cd /data/harbor-installer/harbor
/usr/local/bin/docker-compose stop

cd /backup

delete_and_create_dir
launch_db
wait_for_db_ready
backup_database
backup_redis

# Reiniciar servicios de Harbor lo antes posible
cd /data/harbor-installer/harbor
/usr/local/bin/docker-compose start

# Respaldar otros datos luego de levantar servicios
cd /backup
if [ $dbonly = false ]; then
    backup_registry
    backup_chart_museum
    wait $registry_pid
    wait $chartmuseum_pid
fi
backup_secret

create_tarball
TAR_EXIT_CODE=$?

clean_db

# Mensaje final
if [[ $TAR_EXIT_CODE -ne 0 ]]; then
    echo "Advertencia: La compresión del backup falló, pero Harbor fue restaurado correctamente."
else
    echo "Se respaldaron todos los datos de Harbor. El archivo de backup es $backup_filename."
fi
