#!/bin/bash
NAME=ldap_server
docker rm -f $NAME 2>/dev/null

## Generate cert by IP SAN
## copy server.crt to /data/cert/server.crt
## copy server.key to /data/cert/server.key
## copy harbor_ca.crt to /data/cert/harbor_ca.crt
## After launch ldap server, connect to ldap server by ldaps://<IP>:636
docker run \
	--env LDAP_ORGANISATION="Harbor." \
    --env LDAP_DOMAIN="example.com" \
    --env LDAP_ADMIN_PASSWORD="admin" \
    --env LDAP_TLS_VERIFY_CLIENT="never" \
	--volume /data/cert:/container/service/slapd/assets/certs \
	--env LDAP_TLS_CRT_FILENAME=server.crt \
	--env LDAP_TLS_KEY_FILENAME=server.key \
	--env LDAP_TLS_CA_CRT_FILENAME=harbor_ca.crt \
    -p 389:389 \
    -p 636:636 \
	--detach --name $NAME osixia/openldap:latest


sleep 5
docker cp ldap_test.ldif ldap_server:/
docker exec ldap_server ldapadd -x -D "cn=admin,dc=example,dc=com" -w admin -f /ldap_test.ldif -ZZ

# failed and retry
for number in {1..10}
do
    if [ ! $? -eq 0 ]; then
        sleep 6
        echo "retry in $number "
        docker exec ldap_server ldapadd -x -D "cn=admin,dc=example,dc=com" -w admin -f /ldap_test.ldif -ZZ
    else 
        exit 0
    fi
done
exit 1