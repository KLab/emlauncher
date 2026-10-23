# docker/web/Dockerfile を元にした E2E テスト用イメージ.
# ベースを bookworm に固定し, メールをファイルに書き出すようにしている.
FROM php:8.2.26-apache-bookworm

RUN set -x \
  && apt-get update \
  && apt-get install -y --no-install-recommends \
    unzip pkg-config libssl-dev libsasl2-dev libmagickwand-dev libmemcached-dev libonig-dev libzip-dev zlib1g-dev default-jre-headless curl \
  && docker-php-ext-install -j$(nproc) pdo_mysql zip \
  && pecl install imagick-3.7.0 \
  && pecl install --configureoptions 'with-libmemcached-dir="/usr" with-zlib-dir="/usr" enable-memcached-session="yes" enable-memcached-sasl="no"' memcached-3.3.0 \
  && docker-php-ext-enable imagick memcached \
  && a2enmod rewrite \
  && rm -rf /var/lib/apt/lists/*

RUN set -x \
  && curl -sSfL -o /bundletool.jar https://github.com/google/bundletool/releases/download/1.17.2/bundletool-all-1.17.2.jar \
  && keytool -genkeypair -keystore /emlauncher.keystore -alias emlauncher -storepass emlauncher -keypass emlauncher \
       -keyalg RSA -keysize 2048 -validity 36524 -dname 'CN=EMLauncher E2E, O=KLab Inc., L=Tokyo, C=JP' \
  && chmod 644 /emlauncher.keystore \
  && echo 'emlauncher:emlauncher' > /dbauth \
  && mkdir -p /var/www/emlauncher /mailbox

COPY php.ini /usr/local/etc/php/conf.d/zz-e2e.ini
COPY capture-mail /usr/local/bin/capture-mail
COPY 000-default.conf /etc/apache2/sites-enabled/000-default.conf
RUN chmod 755 /usr/local/bin/capture-mail
