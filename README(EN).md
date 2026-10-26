EMLauncher
==========

An in-house app distribution service. Upload iOS (ipa) / Android (apk, aab) packages and install them on devices.

EMLauncher is implemented in Go. The PHP version has been removed from this repository (the last PHP version is commit `4124ca6`). To migrate from the PHP version, see [docs/go-migration/MIGRATION.md](docs/go-migration/MIGRATION.md) (in Japanese).

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
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o dist/ ./cmd/emlauncher ./cmd/delete-old-packages
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

### 4. Configuration

Copy [config.sample.yml](config.sample.yml) and edit it for your environment (e.g. `/etc/emlauncher/config.yml`). Main items:

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
ExecStart=/opt/emlauncher/emlauncher -config /etc/emlauncher/config.yml
Restart=always

[Install]
WantedBy=multi-user.target
```

EMLauncher listens on `listen` (default `127.0.0.1:8080`). Put it behind a reverse proxy (Apache / nginx) or an ALB. URLs are built from `X-Forwarded-Proto` / `X-Forwarded-Host`. The health check path is `/health`.

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

- Tests: `go test ./...`. Tests using the DB or S3 run only when the E2E environment is up and `EML_TEST_DB_HOST` etc. are set.
- E2E tests and migration rehearsal: [docs/go-migration/TESTING.md](docs/go-migration/TESTING.md) (in Japanese)
