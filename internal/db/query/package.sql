-- name: GetPackage :one
SELECT * FROM package WHERE id = ?;

-- name: GetPackageForUpdate :one
SELECT * FROM package WHERE id = ? FOR UPDATE;

-- name: GetPackagesByIDs :many
SELECT * FROM package WHERE id IN (sqlc.slice('ids'));

-- name: GetNewestPackage :one
SELECT * FROM package WHERE app_id = ? ORDER BY id DESC LIMIT 1;

-- name: InsertPackage :execlastid
INSERT INTO package (app_id, platform, file_name, title, description, identifier, original_file_name, file_size, protect, created)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdatePackageInfo :exec
UPDATE package SET title = ?, description = ?, protect = ? WHERE id = ?;

-- name: DeletePackage :exec
DELETE FROM package WHERE id = ?;

-- name: ListPackages :many
-- platform が NULL なら絞り込まない.
SELECT * FROM package
WHERE app_id = sqlc.arg('app_id')
  AND (sqlc.narg('platform') IS NULL OR platform = sqlc.narg('platform'))
ORDER BY id DESC
LIMIT ? OFFSET ?;

-- name: ListPackagesWithTags :many
-- tag_ids のタグをすべて持つもの. tag_count は tag_ids の (重複の無い) 数.
SELECT p.* FROM package p
JOIN package_tag j ON j.package_id = p.id
WHERE p.app_id = sqlc.arg('app_id')
  AND (sqlc.narg('platform') IS NULL OR p.platform = sqlc.narg('platform'))
  AND j.tag_id IN (sqlc.slice('tag_ids'))
GROUP BY p.id
HAVING count(*) = CAST(sqlc.arg('tag_count') AS SIGNED)
ORDER BY p.id DESC
LIMIT ? OFFSET ?;

-- name: ListUnprotectedPackageIDs :many
-- 保護されていないパッケージの id (新しい順). 古いパッケージの削除に使う.
SELECT id FROM package WHERE app_id = ? AND protect = 0 ORDER BY id DESC;

-- name: ListInstalledPackages :many
-- ユーザがインストールしたことのある, アプリのパッケージ (新しい順).
SELECT p.* FROM package p
WHERE p.app_id = ?
  AND EXISTS (SELECT 1 FROM install_log l WHERE l.package_id = p.id AND l.mail = ?)
ORDER BY p.id DESC;

-- name: ListAttachedFiles :many
SELECT * FROM attached_file WHERE package_id = ? ORDER BY id;

-- name: ListAttachedFilesByPackages :many
SELECT * FROM attached_file WHERE package_id IN (sqlc.slice('package_ids')) ORDER BY package_id, id;

-- name: GetAttachedFile :one
SELECT * FROM attached_file WHERE id = ?;

-- name: InsertAttachedFile :execlastid
INSERT INTO attached_file (app_id, package_id, file_name, original_file_name, file_size, file_type, created)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: DeleteAttachedFile :exec
DELETE FROM attached_file WHERE id = ?;

-- name: DeleteAttachedFilesByPackage :exec
DELETE FROM attached_file WHERE package_id = ?;
