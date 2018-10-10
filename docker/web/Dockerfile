FROM php:7.2-apache
ENV APACHE_DOCUMENT_ROOT /repo/web

RUN \
apt-get update && \
docker-php-ext-install pdo_mysql mysqli mbstring && \
a2enmod rewrite
COPY 000-default.conf /etc/apache2/sites-enabled/000-default.conf
RUN \
echo 'emlauncher:emlauncher' > /dbauth && \
sed -ri -e 's!/var/www/html!${APACHE_DOCUMENT_ROOT}!g' /etc/apache2/sites-available/*.conf && \
sed -ri -e 's!/var/www/!${APACHE_DOCUMENT_ROOT}!g' /etc/apache2/apache2.conf /etc/apache2/conf-available/*.conf

