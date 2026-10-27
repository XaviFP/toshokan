# A new major version can't start on an older one's data directory, so the major is pinned.
FROM postgres:18-bookworm
COPY create-multiple-postgresql-databases.sh /docker-entrypoint-initdb.d/
