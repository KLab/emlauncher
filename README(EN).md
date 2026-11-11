EMLauncher
==========

An in-house app distribution service. Upload iOS (ipa) / Android (apk, aab) packages and install them on devices.

EMLauncher is implemented in Go. The PHP version has been removed from this repository (the last PHP version is commit `4124ca6`). To migrate from the PHP version, see [docs/migration-from-php.md](docs/migration-from-php.md) (in Japanese).

## Requirements

- MySQL 5.7+ / MariaDB
- memcached
- Storage: Amazon S3 (or S3 compatible storage), or local disk
- Java (for bundletool and keytool, to build apk from Android App Bundle (aab))
- A sendmail compatible command (notification and password reset mails)
- Go 1.23+ to build

## Setup

### 1. Build

```sh
cd src && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ../dist/ ./cmd/emlauncher ./cmd/delete-old-packages
```

Deploy `dist/emlauncher` (HTTP server) and `dist/delete-old-packages` (batch to delete old packages) to the server. Templates and CSS are embedded in the binary.

### 2. Database

Edit the password in [sql/database.sql](sql/database.sql), then create the database and tables.

```sh
mysql -uroot < sql/database.sql
mysql -uroot emlauncher < sql/tables.sql
```

### 3. bundletool (for Android App Bundle)

```sh
curl -sLO https://github.com/google/bundletool/releases/download/1.17.2/bundletool-all-1.17.2.jar
keytool -genkeypair -keystore emlauncher-keystore.jks -keyalg RSA -keysize 2048 -validity 10000 -alias emlauncherkey
```

Set the keystore password, file name and alias in `apkfile` of the config file.
On ARM64 servers, the aapt2 bundled in bundletool does not work. Prepare an ARM64 aapt2 and set it to `apkfile.aapt2`.
bundletool extracts its aapt2 into the Java temporary directory (default `/tmp`) and executes it. If `/tmp` is mounted with noexec, add `-Djava.io.tmpdir=<an executable directory>` to `apkfile.java` (e.g. `java -Xmx2048m -Djava.io.tmpdir=/var/tmp`).

### 4. Configuration

Copy [src/config.sample.yml](src/config.sample.yml) and edit it for your environment (e.g. `/etc/emlauncher/config.yml`). Main items:

- `timezone`: time zone of datetimes in the DB (empty: the environment's time zone)
- `database`, `memcache`
- `storage_class`: `S3` / `LocalFile`. Create the S3 bucket in advance
- `session.secret`: a long random string. Use the same value on all servers
- `login`: password login, Google / AzureAD login

### 5. Run

Example systemd unit:

```ini
[Unit]
Description=EMLauncher
After=network.target

[Service]
User=emlauncher
RuntimeDirectory=emlauncher
PIDFile=/run/emlauncher/emlauncher.pid
ExecStart=/opt/emlauncher/emlauncher -config /etc/emlauncher/config.yml -pid-file /run/emlauncher/emlauncher.pid
ExecReload=/bin/sh -c 'kill -HUP $MAINPID'
# Wait for in-flight requests (large uploads) on stop. Match -drain-timeout of emlauncher (default 1h)
TimeoutStopSec=1h
Restart=always

[Install]
WantedBy=multi-user.target
```

EMLauncher listens on `listen` (default `127.0.0.1:8080`). Put it behind a reverse proxy (Apache / nginx) or an ALB. URLs are built from `X-Forwarded-Proto` / `X-Forwarded-Host`. The health check path is `/health`.

#### Update (zero-downtime restart)

Replace the binary or the config file and run `systemctl reload emlauncher`. A new process takes over the listening socket, and the old process exits after finishing in-flight requests (including long uploads). If the new process fails to start (e.g. a config error), the old process keeps running (`upgrade failed` in the log).

`systemctl restart` stops the service first, so it waits for in-flight uploads and the service is down meanwhile. Use `reload` to update.

Uploaded files are written to the temporary directory (`TMPDIR`, default `/tmp`) first. Make sure it has enough space for the largest package.

### 6. Add users

For password login, register only the mail address in the `user_pass` table, and let the user set the password via "forgot password" on the login page.

```sql
INSERT INTO user_pass (mail) VALUES ('your-name@example.com');
```

For Google / AzureAD login, accounts matching `login.allowed_mailaddr_pattern` or registered in `user_pass` can log in.

### 7. Delete old packages (optional)

Deletes unprotected packages, keeping the specified number of newest packages per application. Run it from cron etc.

```sh
/opt/emlauncher/delete-old-packages -config /etc/emlauncher/config.yml -k 100
```

## Development

- Tests: `cd src && go test ./...`. Tests using the DB or S3 run only when the E2E environment is up and `EML_TEST_DB_HOST` etc. are set.
- E2E tests and migration rehearsal: [e2e/README.md](e2e/README.md) (in Japanese)
- Design notes: [docs/design.md](docs/design.md) (in Japanese)
