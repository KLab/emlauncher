-- name: GetApplication :one
SELECT * FROM application WHERE id = ?;

-- name: GetApplicationForUpdate :one
-- アプリのパッケージ・タグ・コメント・オーナーを変更するときは, この行ロックで直列化する.
SELECT * FROM application WHERE id = ? FOR UPDATE;

-- name: GetApplicationByAPIKey :one
SELECT * FROM application WHERE api_key = ?;

-- name: APIKeyExists :one
SELECT EXISTS(SELECT 1 FROM application WHERE api_key = ?) AS found;

-- name: CountApplications :one
SELECT count(*) FROM application;

-- name: ListApplicationsByDateToSort :many
SELECT * FROM application ORDER BY date_to_sort DESC LIMIT ? OFFSET ?;

-- name: ListApplicationIDs :many
SELECT id FROM application ORDER BY id;

-- name: ListApplicationsByOwner :many
SELECT a.* FROM application a
JOIN application_owner o ON o.app_id = a.id
WHERE o.owner_mail = ?
ORDER BY a.id DESC;

-- name: GetApplicationsByIDs :many
SELECT * FROM application WHERE id IN (sqlc.slice('ids'));

-- name: InsertApplication :execlastid
INSERT INTO application (title, api_key, description, repository, created, date_to_sort)
VALUES (?, ?, ?, ?, ?, ?);

-- name: UpdateApplicationIcon :exec
UPDATE application SET icon_key = ? WHERE id = ?;

-- name: UpdateApplicationInfo :exec
UPDATE application SET title = ?, description = ?, repository = ?, icon_key = ? WHERE id = ?;

-- name: UpdateApplicationAPIKey :exec
UPDATE application SET api_key = ? WHERE id = ?;

-- name: UpdateApplicationLastUpload :exec
UPDATE application SET last_upload = ?, date_to_sort = ? WHERE id = ?;

-- name: UpdateApplicationLastCommented :exec
UPDATE application SET last_commented = ?, date_to_sort = ? WHERE id = ?;

-- name: ListOwners :many
SELECT owner_mail FROM application_owner WHERE app_id = ? ORDER BY id;

-- name: IsOwner :one
SELECT EXISTS(SELECT 1 FROM application_owner WHERE app_id = ? AND owner_mail = ?) AS owner;

-- name: InsertOwner :exec
INSERT INTO application_owner (app_id, owner_mail) VALUES (?, ?);

-- name: DeleteOwner :exec
DELETE FROM application_owner WHERE app_id = ? AND owner_mail = ?;
