package university

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UniversityRepository interface {
	Create(ctx context.Context, req *CreateUniversityRequest) (*University, error)
}

type universityRepository struct {
	db *pgxpool.Pool
}

func NewUniversityRepository(db *pgxpool.Pool) UniversityRepository {
	return &universityRepository{
		db: db,
	}
}

func (r *universityRepository) Create(ctx context.Context, req *CreateUniversityRequest) (*University, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	var uni University

	query := `
	INSERT INTO universities (
		name, slug, overview, excerpt,
		country, state, city, full_location,
		cover_image, logo,
		institution_type, campus_setting,
		in_state_tuition, out_of_state_tuition, international_tuition,
		need_based_aid, merit_scholarships, work_study, no_application_fee,
		acceptance_rate, testing_policy, sat_range, act_range,
		on_campus_housing, freshmen_required_on_campus,
		contact_email, contact_phone, website
	)
	VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
		$11, $12, $13, $14, $15, $16, $17, $18, $19,
		$20, $21, $22, $23, $24, $25, $26, $27, $28
	)
	RETURNING 
		id, name, slug, overview, excerpt,
		country, state, city, full_location,
		cover_image, logo,
		institution_type, campus_setting,
		in_state_tuition, out_of_state_tuition, international_tuition,
		need_based_aid, merit_scholarships, work_study, no_application_fee,
		acceptance_rate, testing_policy, sat_range, act_range,
		on_campus_housing, freshmen_required_on_campus,
		contact_email, contact_phone, website,
		created_at, updated_at
	`

	err = tx.QueryRow(ctx, query,
		req.Name, req.Slug, req.Overview, req.Excerpt,
		req.Country, req.State, req.City, req.FullLocation,
		req.CoverImage, req.Logo,
		req.InstitutionType, req.CampusSetting,
		req.InStateTuition, req.OutOfStateTuition, req.InternationalTuition,
		req.NeedBasedAid, req.MeritScholarships, req.WorkStudy, req.NoApplicationFee,
		req.AcceptanceRate, req.TestingPolicy, req.SatRange, req.ActRange,
		req.OnCampusHousing, req.FreshmenRequiredOnCampus,
		req.ContactEmail, req.ContactPhone, req.Website,
	).Scan(
		&uni.ID, &uni.Name, &uni.Slug, &uni.Overview, &uni.Excerpt,
		&uni.Country, &uni.State, &uni.City, &uni.FullLocation,
		&uni.CoverImage, &uni.Logo,
		&uni.InstitutionType, &uni.CampusSetting,
		&uni.InStateTuition, &uni.OutOfStateTuition, &uni.InternationalTuition,
		&uni.NeedBasedAid, &uni.MeritScholarships, &uni.WorkStudy, &uni.NoApplicationFee,
		&uni.AcceptanceRate, &uni.TestingPolicy, &uni.SatRange, &uni.ActRange,
		&uni.OnCampusHousing, &uni.FreshmenRequiredOnCampus,
		&uni.ContactEmail, &uni.ContactPhone, &uni.Website,
		&uni.CreatedAt, &uni.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	insertBatch := func(table, column string, ids []string) error {
		if len(ids) == 0 {
			return nil
		}

		values := make([]string, 0, len(ids))
		args := make([]interface{}, 0, len(ids)+1)
		args = append(args, uni.ID)

		for i, id := range ids {
			values = append(values, fmt.Sprintf("($1, $%d)", i+2))
			args = append(args, id)
		}

		q := fmt.Sprintf(
			"INSERT INTO %s (university_id, %s) VALUES %s",
			table,
			column,
			strings.Join(values, ","),
		)

		_, err := tx.Exec(ctx, q, args...)
		return err
	}

	if err = insertBatch("university_degree_levels", "degree_level_id", req.DegreeLevelIDs); err != nil {
		return nil, err
	}

	if err = insertBatch("university_majors", "major_id", req.MajorIDs); err != nil {
		return nil, err
	}

	if err = insertBatch("university_study_formats", "study_format_id", req.StudyFormatIDs); err != nil {
		return nil, err
	}

	if err = insertBatch("university_special_affiliations", "special_affiliation_id", req.SpecialAffiliationIDs); err != nil {
		return nil, err
	}

	if err = insertBatch("university_athletics", "athletic_id", req.AthleticIDs); err != nil {
		return nil, err
	}

	if err = insertBatch("university_support_services", "support_service_id", req.SupportServiceIDs); err != nil {
		return nil, err
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &uni, nil
}
