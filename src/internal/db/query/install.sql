-- name: InsertInstallLog :exec
INSERT INTO install_log (app_id, package_id, mail, user_agent, installed) VALUES (?, ?, ?, ?, ?);

-- name: UpsertAppInstallUser :exec
INSERT INTO app_install_user (app_id, mail, last_installed) VALUES (?, ?, sqlc.arg('installed'))
ON DUPLICATE KEY UPDATE last_installed = sqlc.arg('installed');

-- name: ListUserInstallLogs :many
-- ユーザのアプリのインストール履歴 (パッケージごとの最後のインストール日時を求める).
SELECT package_id, installed FROM install_log WHERE app_id = ? AND mail = ?;

-- name: ListPackageInstallUsers :many
SELECT DISTINCT mail FROM install_log WHERE package_id = ? ORDER BY mail;

-- name: CountPackageInstallUsers :many
SELECT package_id, count(DISTINCT mail) AS users FROM install_log
WHERE package_id IN (sqlc.slice('package_ids')) GROUP BY package_id;

-- name: ListAppInstallUsers :many
SELECT * FROM app_install_user WHERE app_id = ? ORDER BY mail;

-- name: CountAppInstallUsers :many
SELECT app_id, count(*) AS users FROM app_install_user
WHERE app_id IN (sqlc.slice('app_ids')) GROUP BY app_id;

-- name: ListInstalledApps :many
SELECT * FROM app_install_user WHERE mail = ? ORDER BY app_id DESC;

-- name: GetAppInstallUser :one
SELECT * FROM app_install_user WHERE app_id = ? AND mail = ?;

-- name: UpdateNotify :exec
UPDATE app_install_user SET notify = ? WHERE app_id = ? AND mail = ?;

-- name: DeleteAppInstallUser :exec
DELETE FROM app_install_user WHERE app_id = ? AND mail = ?;
