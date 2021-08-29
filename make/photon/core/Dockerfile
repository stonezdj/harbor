ARG harbor_base_image_version
ARG harbor_base_namespace
FROM ${harbor_base_namespace}/harbor-core-base:${harbor_base_image_version}


RUN tdnf install -y python3 python3-pip.noarch >> /dev/null 
RUN pip3 install supervisor && pip3 install --upgrade setuptools

COPY ./make/photon/common/install_cert.sh /harbor/
COPY ./make/photon/core/entrypoint.sh /harbor/
COPY ./make/photon/core/harbor_core /harbor/
COPY ./src/core/views /harbor/views
COPY ./make/migrations /harbor/migrations
COPY ./icons /harbor/icons
COPY ./make/photon/core/supervisord.conf /etc/supervisord.conf

RUN chown -R harbor:harbor /etc/pki/tls/certs \
    && chown -R harbor:harbor /harbor/ \
    && chmod u+x /harbor/entrypoint.sh \
    && chmod u+x /harbor/install_cert.sh \
    && chmod u+x /harbor/harbor_core

ENTRYPOINT ["/usr/bin/supervisord", "-c", "/etc/supervisord.conf"]
