-- name: InsertGuestPass :execlastid
INSERT INTO guest_pass (app_id, package_id, mail, token, created, expired) VALUES (?, ?, ?, ?, ?, ?);

-- name: GetGuestPass :one
SELECT * FROM guest_pass WHERE id = ?;

-- name: GetGuestPassByToken :one
SELECT * FROM guest_pass WHERE token = ? LIMIT 1;

-- name: ListActiveGuestPassesByPackage :many
SELECT g.*, (SELECT count(*) FROM guestpass_log l WHERE l.guest_pass_id = g.id) AS install_count
FROM guest_pass g
WHERE g.package_id = ? AND g.expired > sqlc.arg('now')
ORDER BY g.id;

-- name: ListActiveGuestPassesByOwner :many
-- ユーザがオーナーのアプリの, 有効なゲストパス (期限の近い順).
SELECT g.*, a.title AS app_title, a.icon_key AS app_icon_key,
  (SELECT count(*) FROM guestpass_log l WHERE l.guest_pass_id = g.id) AS install_count
FROM guest_pass g
JOIN application_owner o ON o.app_id = g.app_id
JOIN application a ON a.id = g.app_id
WHERE o.owner_mail = ? AND g.expired > sqlc.arg('now')
ORDER BY g.expired, g.id;

-- name: ExpireGuestPass :exec
UPDATE guest_pass SET expired = ? WHERE id = ?;

-- name: InsertGuestPassLog :exec
INSERT INTO guestpass_log (guest_pass_id, user_agent, ip_address, installed) VALUES (?, ?, ?, ?);
