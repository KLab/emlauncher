-- name: CountComments :one
SELECT count(*) FROM comment WHERE app_id = ?;

-- name: CountCommentsByApps :many
SELECT app_id, count(*) AS comments FROM comment
WHERE app_id IN (sqlc.slice('app_ids')) GROUP BY app_id;

-- name: ListComments :many
SELECT * FROM comment WHERE app_id = ? ORDER BY id DESC LIMIT ? OFFSET ?;

-- name: MaxCommentNumber :one
SELECT CAST(COALESCE(max(number), 0) AS SIGNED) AS number FROM comment WHERE app_id = ?;

-- name: InsertComment :execlastid
INSERT INTO comment (app_id, package_id, number, mail, message, created) VALUES (?, ?, ?, ?, ?, ?);
