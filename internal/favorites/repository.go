package favorites

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"fmu-backend/internal/college"
	"fmu-backend/internal/db/sqlc"
	"fmu-backend/internal/pagination"
	"fmu-backend/internal/university"
)

type Repository interface {
	AddUniversity(ctx context.Context, userID, universityID string) error
	RemoveUniversity(ctx context.Context, userID, universityID string) error
	ListUniversities(ctx context.Context, userID string, q pagination.Query) ([]university.UniversityListItem, int64, error)
	FavoritedUniversityIDs(ctx context.Context, userID string, ids []string) (map[string]struct{}, error)

	AddCollege(ctx context.Context, userID, collegeID string) error
	RemoveCollege(ctx context.Context, userID, collegeID string) error
	ListColleges(ctx context.Context, userID string, q pagination.Query) ([]college.CollegeListItem, int64, error)
	FavoritedCollegeIDs(ctx context.Context, userID string, ids []string) (map[string]struct{}, error)
}

type repository struct {
	queries *sqlc.Queries
	pool    *pgxpool.Pool
}

func NewRepository(queries *sqlc.Queries, pool *pgxpool.Pool) Repository {
	return &repository{queries: queries, pool: pool}
}

func (r *repository) AddUniversity(ctx context.Context, userID, universityID string) error {
	return r.queries.AddUniversityFavorite(ctx, sqlc.AddUniversityFavoriteParams{
		UserID:       userID,
		UniversityID: universityID,
	})
}

func (r *repository) RemoveUniversity(ctx context.Context, userID, universityID string) error {
	return r.queries.RemoveUniversityFavorite(ctx, sqlc.RemoveUniversityFavoriteParams{
		UserID:       userID,
		UniversityID: universityID,
	})
}

// listUniversitiesSQL projects only the columns UniversityListItem needs and
// COALESCEs nullable fields to plain strings/scalars so we can scan directly
// into the struct. ORDER BY uf.created_at DESC gives newest-favorited first.
const listUniversitiesSQL = `
SELECT
    u.id,
    u.name,
    u.slug,
    COALESCE(u.country, '')                AS country,
    COALESCE(u.state, '')                  AS state,
    COALESCE(u.city, '')                   AS city,
    COALESCE(u.logo, '')                   AS logo,
    COALESCE(u.cover_image, '')            AS cover_image,
    COALESCE(u.institution_type, '')       AS institution_type,
    COALESCE(u.campus_setting, '')         AS campus_setting,
    COALESCE(u.tuition_min, 0)             AS tuition_min,
    COALESCE(u.tuition_max, 0)             AS tuition_max,
    COALESCE(u.acceptance_rate, 0)::float8 AS acceptance_rate,
    u.is_popular,
    u.is_featured
FROM university_favorites uf
JOIN universities u ON u.id = uf.university_id
WHERE uf.user_id = $1
ORDER BY uf.created_at DESC
LIMIT $2 OFFSET $3
`

func (r *repository) ListUniversities(ctx context.Context, userID string, q pagination.Query) ([]university.UniversityListItem, int64, error) {
	total, err := r.queries.CountFavoritedUniversities(ctx, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("count favorited universities: %w", err)
	}

	rows, err := r.pool.Query(ctx, listUniversitiesSQL, userID, q.Limit(), q.Offset())
	if err != nil {
		return nil, 0, fmt.Errorf("list favorited universities: %w", err)
	}
	defer rows.Close()

	items := []university.UniversityListItem{}
	for rows.Next() {
		var u university.UniversityListItem
		if err := rows.Scan(
			&u.ID, &u.Name, &u.Slug,
			&u.Country, &u.State, &u.City, &u.Logo,
			&u.CoverImage, &u.InstitutionType, &u.CampusSetting,
			&u.TuitionMin, &u.TuitionMax, &u.AcceptanceRate,
			&u.IsPopular, &u.IsFeatured,
		); err != nil {
			return nil, 0, fmt.Errorf("scan favorited university: %w", err)
		}
		items = append(items, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate favorited universities: %w", err)
	}
	return items, total, nil
}

// FavoritedUniversityIDs returns a set of university IDs from the input slice
// that the user has favorited. Used to stamp `is_favorited` on list/search
// responses without an N+1 query.
func (r *repository) FavoritedUniversityIDs(ctx context.Context, userID string, ids []string) (map[string]struct{}, error) {
	if len(ids) == 0 {
		return map[string]struct{}{}, nil
	}
	rows, err := r.queries.ListFavoritedUniversityIDs(ctx, sqlc.ListFavoritedUniversityIDsParams{
		UserID: userID,
		Column2: ids,
	})
	if err != nil {
		return nil, fmt.Errorf("list favorited university ids: %w", err)
	}
	set := make(map[string]struct{}, len(rows))
	for _, id := range rows {
		set[id] = struct{}{}
	}
	return set, nil
}

func (r *repository) AddCollege(ctx context.Context, userID, collegeID string) error {
	return r.queries.AddCollegeFavorite(ctx, sqlc.AddCollegeFavoriteParams{
		UserID:    userID,
		CollegeID: collegeID,
	})
}

func (r *repository) RemoveCollege(ctx context.Context, userID, collegeID string) error {
	return r.queries.RemoveCollegeFavorite(ctx, sqlc.RemoveCollegeFavoriteParams{
		UserID:    userID,
		CollegeID: collegeID,
	})
}

const listCollegesSQL = `
SELECT
    c.id,
    c.name,
    c.slug,
    c.university_id,
    COALESCE(c.country, '') AS country,
    COALESCE(c.state, '')   AS state,
    COALESCE(c.city, '')    AS city,
    COALESCE(c.logo, '')    AS logo
FROM college_favorites cf
JOIN colleges c ON c.id = cf.college_id
WHERE cf.user_id = $1
ORDER BY cf.created_at DESC
LIMIT $2 OFFSET $3
`

func (r *repository) ListColleges(ctx context.Context, userID string, q pagination.Query) ([]college.CollegeListItem, int64, error) {
	total, err := r.queries.CountFavoritedColleges(ctx, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("count favorited colleges: %w", err)
	}

	rows, err := r.pool.Query(ctx, listCollegesSQL, userID, q.Limit(), q.Offset())
	if err != nil {
		return nil, 0, fmt.Errorf("list favorited colleges: %w", err)
	}
	defer rows.Close()

	items := []college.CollegeListItem{}
	for rows.Next() {
		var c college.CollegeListItem
		if err := rows.Scan(
			&c.ID, &c.Name, &c.Slug, &c.UniversityID,
			&c.Country, &c.State, &c.City, &c.Logo,
		); err != nil {
			return nil, 0, fmt.Errorf("scan favorited college: %w", err)
		}
		items = append(items, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate favorited colleges: %w", err)
	}
	return items, total, nil
}

// FavoritedCollegeIDs returns a set of college IDs from the input slice that
// the user has favorited.
func (r *repository) FavoritedCollegeIDs(ctx context.Context, userID string, ids []string) (map[string]struct{}, error) {
	if len(ids) == 0 {
		return map[string]struct{}{}, nil
	}
	rows, err := r.queries.ListFavoritedCollegeIDs(ctx, sqlc.ListFavoritedCollegeIDsParams{
		UserID:   userID,
		Column2: ids,
	})
	if err != nil {
		return nil, fmt.Errorf("list favorited college ids: %w", err)
	}
	set := make(map[string]struct{}, len(rows))
	for _, id := range rows {
		set[id] = struct{}{}
	}
	return set, nil
}