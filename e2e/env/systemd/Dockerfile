# systemd の下で Go 版を動かす E2E テスト用イメージ (ユニットファイルと systemctl reload の確認用).
# ビルドコンテキストはリポジトリ直下.
FROM golang:1.23.4-bookworm AS build
WORKDIR /src
COPY src/go.mod src/go.sum ./
RUN go mod download
COPY src/ .
RUN CGO_ENABLED=0 go build -o /out/emlauncher ./cmd/emlauncher \
  && CGO_ENABLED=0 go build -o /out/delete-old-packages ./cmd/delete-old-packages

FROM debian:bookworm
RUN set -x \
  && apt-get update \
  && mkdir -p /usr/share/man/man1 \
  && apt-get install -y --no-install-recommends systemd systemd-sysv tzdata curl ca-certificates default-jre-headless \
  && rm -rf /var/lib/apt/lists/* \
  && curl -sSfL -o /bundletool.jar https://github.com/google/bundletool/releases/download/1.17.2/bundletool-all-1.17.2.jar \
  && keytool -genkeypair -keystore /emlauncher.keystore -alias emlauncher -storepass emlauncher -keypass emlauncher \
       -keyalg RSA -keysize 2048 -validity 36524 -dname 'CN=EMLauncher E2E, O=KLab Inc., L=Tokyo, C=JP' \
  && chmod 644 /emlauncher.keystore \
  && ln -sf /usr/share/zoneinfo/Asia/Tokyo /etc/localtime \
  && printf '#!/bin/sh\n# sendmail の代わりに, 送信されたメールを /mailbox にファイルとして保存する.\ncat > "/mailbox/$(date +%%s%%N)-$$.eml"\n' > /usr/local/bin/capture-mail \
  && chmod 755 /usr/local/bin/capture-mail \
  && mkdir -p /mailbox /var/www/emlauncher \
  && systemctl mask getty.target systemd-logind.service systemd-udevd.service
COPY --from=build /out/ /usr/local/bin/
COPY e2e/env/systemd/emlauncher.service /etc/systemd/system/emlauncher.service
RUN systemctl enable emlauncher.service
STOPSIGNAL SIGRTMIN+3
CMD ["/lib/systemd/systemd"]
