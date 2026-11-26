-- name: ListTagsByApp :many
SELECT * FROM tag WHERE app_id = ? ORDER BY id;

-- name: ListUnusedTagIDsByApp :many
SELECT id FROM tag WHERE app_id = ? AND id NOT IN (SELECT tag_id FROM package_tag) ORDER BY id;

-- name: ListTagsByPackage :many
SELECT t.* FROM package_tag j JOIN tag t ON t.id = j.tag_id
WHERE j.package_id = ? ORDER BY j.tag_id;

-- name: ListTagsByPackages :many
SELECT j.package_id, t.id, t.name FROM package_tag j JOIN tag t ON t.id = j.tag_id
WHERE j.package_id IN (sqlc.slice('package_ids')) ORDER BY j.package_id, j.tag_id;

-- name: InsertTag :execlastid
INSERT INTO tag (app_id, name) VALUES (?, ?);

-- name: DeletePackageTagsByTagIDs :exec
DELETE FROM package_tag WHERE tag_id IN (sqlc.slice('tag_ids'));

-- name: DeleteTagsByIDs :exec
DELETE FROM tag WHERE app_id = ? AND id IN (sqlc.slice('tag_ids'));

-- name: InsertPackageTag :exec
INSERT INTO package_tag (package_id, tag_id) VALUES (?, ?);

-- name: DeletePackageTags :exec
DELETE FROM package_tag WHERE package_id = ?;
