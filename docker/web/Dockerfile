FROM php:7.2-apache
ENV APACHE_DOCUMENT_ROOT /repo/web

RUN set -x \
  && apt update \
  && apt install -y --no-install-recommends \
    unzip libssl-dev libpcre3 libpcre3-dev zlib1g-dev libmagickwand-dev sendmail

RUN set -x \
  && cd /tmp \
  && curl -sSL -o php7.zip https://github.com/websupport-sk/pecl-memcache/archive/php7.zip \
  && unzip php7 \
  && cd pecl-memcache-php7 \
  && /usr/local/bin/phpize \
  && ./configure --with-php-config=/usr/local/bin/php-config \
  && make \
  && make install \
  && echo "extension=memcache.so" > /usr/local/etc/php/conf.d/ext-memcache.ini \
  && rm -rf /tmp/pecl-memcache-php7 php7.zip

RUN set -x \
  && docker-php-ext-install pdo_mysql mysqli mbstring \
  && pecl install imagick-3.4.3 \
  && docker-php-ext-enable imagick \
  && a2enmod rewrite

COPY entrypoint.sh /entrypoint.sh
COPY 000-default.conf /etc/apache2/sites-enabled/000-default.conf
COPY php.ini /usr/local/etc/php/php.ini

RUN set -x \
  && echo 'emlauncher:emlauncher' > /dbauth \
  && sed -ri -e 's!/var/www/html!${APACHE_DOCUMENT_ROOT}!g' /etc/apache2/sites-available/*.conf \
  && sed -ri -e 's!/var/www/!${APACHE_DOCUMENT_ROOT}!g' /etc/apache2/apache2.conf /etc/apache2/conf-available/*.conf

ENTRYPOINT /entrypoint.sh
