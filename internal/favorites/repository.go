package favorites

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"fmu-backend/internal/college"
	"fmu-backend/internal/db/sqlc"
	"fmu-backend/internal/pagination"
	"fmu-backend/internal/scholarship"
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

	AddScholarship(ctx context.Context, userID, scholarshipID string) error
	RemoveScholarship(ctx context.Context, userID, scholarshipID string) error
	ListScholarships(ctx context.Context, userID string, q pagination.Query) ([]scholarship.ScholarshipListItem, int64, error)
	FavoritedScholarshipIDs(ctx context.Context, userID string, ids []string) (map[string]struct{}, error)
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
    u.is_featured,
    EXISTS (
        SELECT 1 FROM users rep
        WHERE rep.representative_university_id = u.id
    ) AS has_representative
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
			&u.IsPopular, &u.IsFeatured, &u.HasRepresentative,
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
		UserID:  userID,
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
    COALESCE(c.city, '')             AS city,
    COALESCE(c.logo, '')             AS logo,
    COALESCE(c.cover_image, '')      AS cover_image,
    COALESCE(c.institution_type, '') AS institution_type,
    COALESCE(c.campus_setting, '')   AS campus_setting,
    c.is_popular,
    c.is_featured,
    EXISTS (
        SELECT 1 FROM users rep
        WHERE rep.representative_college_id = c.id
    ) AS has_representative
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
			&c.CoverImage, &c.InstitutionType, &c.CampusSetting,
			&c.IsPopular, &c.IsFeatured, &c.HasRepresentative,
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
		UserID:  userID,
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

func (r *repository) AddScholarship(ctx context.Context, userID, scholarshipID string) error {
	return r.queries.AddScholarshipFavorite(ctx, sqlc.AddScholarshipFavoriteParams{
		UserID:        userID,
		ScholarshipID: scholarshipID,
	})
}

func (r *repository) RemoveScholarship(ctx context.Context, userID, scholarshipID string) error {
	return r.queries.RemoveScholarshipFavorite(ctx, sqlc.RemoveScholarshipFavoriteParams{
		UserID:        userID,
		ScholarshipID: scholarshipID,
	})
}

const listScholarshipsSQL = `
SELECT
    s.id,
    s.title,
    s.slug,
    s.award_amount,
    COALESCE(s.logo, '')          AS logo,
    COALESCE(s.provider_type, '') AS provider_type,
    COALESCE(s.provider_name, '') AS provider_name,
    COALESCE(s.country, '')       AS country,
    s.application_deadline,
    COALESCE(s.min_gpa, 0)::float8 AS min_gpa,
    s.requires_financial_need,
    s.essay_required,
    s.is_renewable,
    s.is_popular,
    s.is_featured
FROM scholarship_favorites sf
JOIN scholarships s ON s.id = sf.scholarship_id
WHERE sf.user_id = $1
ORDER BY sf.created_at DESC
LIMIT $2 OFFSET $3
`

func (r *repository) ListScholarships(ctx context.Context, userID string, q pagination.Query) ([]scholarship.ScholarshipListItem, int64, error) {
	total, err := r.queries.CountFavoritedScholarships(ctx, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("count favorited scholarships: %w", err)
	}

	rows, err := r.pool.Query(ctx, listScholarshipsSQL, userID, q.Limit(), q.Offset())
	if err != nil {
		return nil, 0, fmt.Errorf("list favorited scholarships: %w", err)
	}
	defer rows.Close()

	items := []scholarship.ScholarshipListItem{}
	for rows.Next() {
		var s scholarship.ScholarshipListItem
		if err := rows.Scan(
			&s.ID, &s.Title, &s.Slug, &s.AwardAmount,
			&s.Logo, &s.ProviderType, &s.ProviderName, &s.Country,
			&s.ApplicationDeadline, &s.MinGpa,
			&s.RequiresFinancialNeed, &s.EssayRequired, &s.IsRenewable,
			&s.IsPopular, &s.IsFeatured,
		); err != nil {
			return nil, 0, fmt.Errorf("scan favorited scholarship: %w", err)
		}
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate favorited scholarships: %w", err)
	}
	return items, total, nil
}

// FavoritedScholarshipIDs returns a set of scholarship IDs from the input slice
// that the user has favorited.
func (r *repository) FavoritedScholarshipIDs(ctx context.Context, userID string, ids []string) (map[string]struct{}, error) {
	if len(ids) == 0 {
		return map[string]struct{}{}, nil
	}
	rows, err := r.queries.ListFavoritedScholarshipIDs(ctx, sqlc.ListFavoritedScholarshipIDsParams{
		UserID:  userID,
		Column2: ids,
	})
	if err != nil {
		return nil, fmt.Errorf("list favorited scholarship ids: %w", err)
	}
	set := make(map[string]struct{}, len(rows))
	for _, id := range rows {
		set[id] = struct{}{}
	}
	return set, nil
}
