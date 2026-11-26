-- name: GetUserPass :one
SELECT * FROM user_pass WHERE mail = ?;

-- name: UpdatePasshash :exec
UPDATE user_pass SET passhash = ? WHERE mail = ?;
