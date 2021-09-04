ARG harbor_base_image_version
ARG harbor_base_namespace
FROM ${harbor_base_namespace}/harbor-core-base:${harbor_base_image_version}

ENV PGDATA /var/lib/postgresql/data

RUN tdnf install -y python3 python3-pip.noarch gzip postgresql findutils bc procps-ng >> /dev/null \
    && groupadd -r postgres --gid=999 \
    && useradd -m -r -g postgres --uid=999 postgres \
    && mkdir -p /docker-entrypoint-initdb.d \
    && mkdir -p /run/postgresql \
    && chown -R postgres:postgres /run/postgresql \
    && chmod 2777 /run/postgresql \
    && mkdir -p "$PGDATA" && chown -R postgres:postgres "$PGDATA" && chmod 777 "$PGDATA" \
    && sed -i "s|#listen_addresses = 'localhost'.*|listen_addresses = '*'|g" /usr/share/postgresql/postgresql.conf.sample \
    && sed -i "s|#unix_socket_directories = '/tmp'.*|unix_socket_directories = '/run/postgresql'|g" /usr/share/postgresql/postgresql.conf.sample 

RUN tdnf erase -y toybox && tdnf install -y util-linux net-tools redis

RUN pip3 install supervisor && pip3 install --upgrade setuptools

COPY ./make/photon/common/install_cert.sh /harbor/
COPY ./make/photon/core/entrypoint.sh /harbor/
COPY ./make/photon/core/harbor_core /harbor/
COPY ./src/core/views /harbor/views
COPY ./make/migrations /harbor/migrations
COPY ./icons /harbor/icons


VOLUME /var/lib/postgresql/data
VOLUME /var/lib/redis
VOLUME /storage

# prepare core
RUN chown -R harbor:harbor /etc/pki/tls/certs \
    && chown -R harbor:harbor /harbor/ \
    && chmod u+x /harbor/entrypoint.sh \
    && chmod u+x /harbor/install_cert.sh \
    && chmod u+x /harbor/harbor_core

# prepare postgres
COPY ./make/photon/db/docker-entrypoint.sh /postgres-docker-entrypoint.sh
COPY ./make/photon/db/initdb.sh /initdb.sh
COPY ./make/photon/db/upgrade.sh /upgrade.sh
COPY ./make/photon/db/docker-healthcheck.sh /docker-healthcheck.sh
COPY ./make/photon/db/initial-notaryserver.sql /docker-entrypoint-initdb.d/
COPY ./make/photon/db/initial-notarysigner.sql /docker-entrypoint-initdb.d/
COPY ./make/photon/db/initial-registry.sql /docker-entrypoint-initdb.d/
RUN chown -R postgres:postgres /postgres-docker-entrypoint.sh /docker-healthcheck.sh /docker-entrypoint-initdb.d \
    && chmod u+x /postgres-docker-entrypoint.sh /docker-healthcheck.sh

COPY ./make/photon/core/supervisord.conf /etc/supervisord.conf

# prepare redis
COPY ./make/photon/redis/docker-healthcheck /usr/bin/
COPY ./make/photon/redis/redis.conf /etc/redis.conf
RUN chmod +x /usr/bin/docker-healthcheck \
    && chown redis:redis /etc/redis.conf    

# prepare registry and registry ctl
RUN mkdir -p /etc/registry && mkdir -p /etc/registryctl 
COPY ./make/photon/registry/config.yml /etc/registry/config.yml
COPY ./make/photon/registry/passwd   /etc/registry/
COPY ./make/photon/registry/root.crt   /etc/registry/
COPY ./make/photon/common/install_cert.sh /home/harbor
COPY ./make/photon/registry/entrypoint.sh /home/harbor/registry-entrypoint.sh
COPY ./make/photon/registry/binary/registry /usr/bin/registry_DO_NOT_USE_GC
COPY ./make/photon/registryctl/start.sh /home/harbor
COPY ./make/photon/registryctl/harbor_registryctl /home/harbor
COPY ./make/photon/registryctl/config.yml /etc/registryctl/config.yml

RUN chown -R harbor:harbor /etc/pki/tls/certs \
    && chown harbor:harbor /etc/registry/config.yml \
    && chown harbor:harbor /etc/registry/passwd \
    && chown harbor:harbor /etc/registry/root.crt \
    && chown harbor:harbor /home/harbor/install_cert.sh && chmod u+x /home/harbor/install_cert.sh \
    && chown harbor:harbor /home/harbor/registry-entrypoint.sh && chmod u+x /home/harbor/registry-entrypoint.sh \
    && chown harbor:harbor /home/harbor/harbor_registryctl && chmod u+x /home/harbor/harbor_registryctl \
    && chown harbor:harbor /home/harbor/start.sh && chmod u+x /home/harbor/start.sh \
    && chown harbor:harbor /usr/bin/registry_DO_NOT_USE_GC && chmod u+x /usr/bin/registry_DO_NOT_USE_GC \
    && chown harbor:harbor /etc/registryctl/config.yml

ENTRYPOINT ["/usr/bin/supervisord", "-c", "/etc/supervisord.conf"]
