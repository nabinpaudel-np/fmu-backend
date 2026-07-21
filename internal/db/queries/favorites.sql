-- name: AddUniversityFavorite :exec
INSERT INTO university_favorites (user_id, university_id)
VALUES ($1, $2)
ON CONFLICT (user_id, university_id) DO NOTHING;

-- name: RemoveUniversityFavorite :exec
DELETE FROM university_favorites WHERE user_id = $1 AND university_id = $2;

-- name: CountFavoritedUniversities :one
SELECT COUNT(*) FROM university_favorites WHERE user_id = $1;
-- name: AddCollegeFavorite :exec
INSERT INTO college_favorites (user_id, college_id)
VALUES ($1, $2)
ON CONFLICT (user_id, college_id) DO NOTHING;

-- name: RemoveCollegeFavorite :exec
DELETE FROM college_favorites WHERE user_id = $1 AND college_id = $2;

-- name: CountFavoritedColleges :one
SELECT COUNT(*) FROM college_favorites WHERE user_id = $1;

-- name: ListFavoritedUniversityIDs :many
SELECT university_id FROM university_favorites
WHERE user_id = $1 AND university_id = ANY($2::uuid[]);

-- name: ListFavoritedCollegeIDs :many
SELECT college_id FROM college_favorites
WHERE user_id = $1 AND college_id = ANY($2::uuid[]);