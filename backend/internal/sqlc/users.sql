-- name: GetUser :one
select
    *
from users u
where 1=1
    and u.email = sqlc.arg(email)
;
