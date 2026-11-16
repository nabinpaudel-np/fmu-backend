package nlsearch

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"fmu-backend/internal/pagination"
)

// The repository runs a compiled plan. It owns the row shape — which columns
// are read and how they are scanned — but not the filtering: the predicates
// arrive already built by the compiler.

// Repository reads the catalogue for a validated plan.
type Repository interface {
	Search(ctx context.Context, plan *Plan, q pagination.Query) ([]ResultItem, int64, error)
}

type repository struct {
	pool *pgxpool.Pool
}

// NewRepository builds a repository over the catalogue.
func NewRepository(pool *pgxpool.Pool) Repository {
	return &repository{pool: pool}
}

// Search runs the plan and returns the matching rows and the total number of
// matches.
func (r *repository) Search(ctx context.Context, plan *Plan, q pagination.Query) ([]ResultItem, int64, error) {
	entity := plan.Entity
	compiled := Compile(plan)

	where := " WHERE 1=1" + compiled.Where

	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM %s %s%s", entity.Table, entity.Alias, where)
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, compiled.Args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count %s: %w", entity.Name, err)
	}

	args := make([]any, 0, len(compiled.Args)+2)
	args = append(args, compiled.Args...)
	args = append(args, q.Limit(), q.Offset())

	listSQL := fmt.Sprintf(
		"SELECT %s FROM %s %s%s ORDER BY %s LIMIT $%d OFFSET $%d",
		strings.Join(selectColumns(entity), ", "),
		entity.Table, entity.Alias, where, compiled.OrderBy,
		len(compiled.Args)+1, len(compiled.Args)+2,
	)

	rows, err := r.pool.Query(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("search %s: %w", entity.Name, err)
	}
	defer rows.Close()

	items, err := scanItems(entity, rows)
	if err != nil {
		return nil, 0, fmt.Errorf("scan %s: %w", entity.Name, err)
	}
	return items, total, nil
}

// selectColumns lists the columns a result row is built from. They are fixed
// literals — nothing a user writes reaches this list.
func selectColumns(entity *Entity) []string {
	a := entity.Alias + "."
	switch entity.Name {
	case "universities":
		return []string{
			a + "id",
			a + "name",
			a + "slug",
			"COALESCE(" + a + "logo, '') AS logo",
			"COALESCE(" + a + "cover_image, '') AS cover_image",
			"COALESCE(" + a + "country, '') AS country",
			"COALESCE(" + a + "state, '') AS state",
			"COALESCE(" + a + "city, '') AS city",
			"COALESCE(" + a + "full_location, '') AS full_location",
			"COALESCE(" + a + "institution_type, '') AS institution_type",
			"COALESCE(" + a + "campus_setting, '') AS campus_setting",
			"COALESCE(" + a + "tuition_min, 0)::float8 AS tuition_min",
			"COALESCE(" + a + "tuition_max, 0)::float8 AS tuition_max",
			"COALESCE(" + a + "acceptance_rate, 0)::float8 AS acceptance_rate",
			"COALESCE(" + a + "employment_rate, 0)::float8 AS employment_rate",
			"COALESCE(" + a + "founded_year, 0)::int AS founded_year",
			a + "is_popular",
			a + "is_featured",
		}
	case "colleges":
		return []string{
			a + "id",
			a + "name",
			a + "slug",
			"COALESCE(" + a + "logo, '') AS logo",
			"COALESCE(" + a + "cover_image, '') AS cover_image",
			"COALESCE(" + a + "country, '') AS country",
			"COALESCE(" + a + "state, '') AS state",
			"COALESCE(" + a + "city, '') AS city",
			"COALESCE(" + a + "full_location, '') AS full_location",
			"COALESCE(" + a + "institution_type, '') AS institution_type",
			"COALESCE(" + a + "campus_setting, '') AS campus_setting",
			"COALESCE(" + a + "founded_year, 0)::int AS founded_year",
			a + "is_popular",
			a + "is_featured",
		}
	case "scholarships":
		return []string{
			a + "id",
			a + "title",
			a + "slug",
			"COALESCE(" + a + "logo, '') AS logo",
			a + "award_amount",
			"COALESCE(" + a + "award_min, 0)::float8 AS award_min",
			"COALESCE(" + a + "award_max, 0)::float8 AS award_max",
			"COALESCE(" + a + "provider_name, '') AS provider_name",
			"COALESCE(" + a + "provider_type, '') AS provider_type",
			"COALESCE(" + a + "country, '') AS country",
			"COALESCE(" + a + "state, '') AS state",
			"COALESCE(" + a + "city, '') AS city",
			"COALESCE(" + a + "min_gpa, 0)::float8 AS min_gpa",
			"COALESCE(" + a + "number_of_awards, 0)::int AS number_of_awards",
			a + "application_deadline",
			a + "is_popular",
			a + "is_featured",
		}
	case "programs":
		// A programme has no slug and no publication state, so the row is
		// identified by its title alone. The frontend links to a programme by
		// title, so that is what the slug carries here.
		return []string{
			a + "id",
			a + "title",
			a + "title AS slug",
			"'' AS logo",
			"'' AS cover_image",
			"COALESCE(" + a + "description, '') AS description",
			"COALESCE(" + a + "excerpt, '') AS excerpt",
			"COALESCE(" + a + "career_options, '') AS career_options",
			"false AS is_popular",
			"false AS is_featured",
		}
	}
	return []string{a + "id"}
}

// scanItems reads the rows of a result list. The column order here must
// match selectColumns.
func scanItems(entity *Entity, rows pgx.Rows) ([]ResultItem, error) {
	items := make([]ResultItem, 0, 16)
	for rows.Next() {
		var item ResultItem
		var err error
		switch entity.Name {
		case "universities":
			err = rows.Scan(
				&item.ID, &item.Title, &item.Slug,
				&item.Logo, &item.CoverImage,
				&item.Country, &item.State, &item.City, &item.FullLocation,
				&item.InstitutionType, &item.CampusSetting,
				&item.TuitionMin, &item.TuitionMax,
				&item.AcceptanceRate, &item.EmploymentRate, &item.FoundedYear,
				&item.IsPopular, &item.IsFeatured,
			)
		case "colleges":
			err = rows.Scan(
				&item.ID, &item.Title, &item.Slug,
				&item.Logo, &item.CoverImage,
				&item.Country, &item.State, &item.City, &item.FullLocation,
				&item.InstitutionType, &item.CampusSetting,
				&item.FoundedYear,
				&item.IsPopular, &item.IsFeatured,
			)
		case "scholarships":
			err = rows.Scan(
				&item.ID, &item.Title, &item.Slug,
				&item.Logo, &item.AwardAmount,
				&item.AwardMin, &item.AwardMax,
				&item.ProviderName, &item.ProviderType,
				&item.Country, &item.State, &item.City,
				&item.MinGpa, &item.NumberOfAwards,
				&item.ApplicationDeadline,
				&item.IsPopular, &item.IsFeatured,
			)
		case "programs":
			err = rows.Scan(
				&item.ID, &item.Title, &item.Slug,
				&item.Logo, &item.CoverImage,
				&item.Description, &item.Excerpt, &item.CareerOptions,
				&item.IsPopular, &item.IsFeatured,
			)
		default:
			return nil, fmt.Errorf("unknown entity %q", entity.Name)
		}
		if err != nil {
			return nil, err
		}
		item.Entity = entity.Name
		items = append(items, item)
	}
	return items, rows.Err()
}
