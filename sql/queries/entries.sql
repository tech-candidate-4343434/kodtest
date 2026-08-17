-- name: ListEntries :many
-- Newest first. CURRENT_TIMESTAMP only resolves to the second, so entries
-- signed within the same second tie; id breaks the tie in insertion order.
SELECT * FROM entries ORDER BY created_at DESC, id DESC;

-- name: CreateEntry :exec
INSERT INTO entries (name, message) VALUES (?, ?);