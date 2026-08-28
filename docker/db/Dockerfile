FROM mysql/mysql-server:8.0
ENV TZ: Asia/Tokyo

COPY init/00_database.sql /docker-entrypoint-initdb.d/
COPY init/01_tables.sql /docker-entrypoint-initdb.d/
