# E2E テスト用の Go 版イメージ. ビルドコンテキストはリポジトリ直下.
FROM golang:1.23.4-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/emlauncher ./cmd/emlauncher \
  && CGO_ENABLED=0 go build -o /out/delete-old-packages ./cmd/delete-old-packages

FROM debian:bookworm-slim
RUN set -x \
  && mkdir -p /usr/share/man/man1 \
  && apt-get update \
  && apt-get install -y --no-install-recommends default-jre-headless curl ca-certificates \
  && rm -rf /var/lib/apt/lists/* \
  && curl -sSfL -o /bundletool.jar https://github.com/google/bundletool/releases/download/1.17.2/bundletool-all-1.17.2.jar \
  && keytool -genkeypair -keystore /emlauncher.keystore -alias emlauncher -storepass emlauncher -keypass emlauncher \
       -keyalg RSA -keysize 2048 -validity 36524 -dname 'CN=EMLauncher E2E, O=KLab Inc., L=Tokyo, C=JP' \
  && printf '#!/bin/sh\n# sendmail の代わりに, 送信されたメールを /mailbox にファイルとして保存する.\ncat > "/mailbox/$(date +%%s%%N)-$$.eml"\n' > /usr/local/bin/capture-mail \
  && chmod 755 /usr/local/bin/capture-mail \
  && mkdir -p /mailbox /var/www/emlauncher
COPY --from=build /out/ /usr/local/bin/
CMD ["sh", "-c", "exec emlauncher -config /etc/emlauncher/config.${E2E_ENV}.yml"]
