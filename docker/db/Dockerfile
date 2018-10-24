FROM mysql:5.7
ENV TZ: Asia/Tokyo

COPY init/00_database.sql /docker-entrypoint-initdb.d/
COPY init/01_tables.sql /docker-entrypoint-initdb.d/
