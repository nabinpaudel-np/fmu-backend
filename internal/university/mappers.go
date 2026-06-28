package university

import (
	"github.com/jackc/pgx/v5/pgtype"

	"fmu-backend/internal/db/sqlc"
)

func toCreateUniversityParams(req *CreateUniversityRequest) sqlc.CreateUniversityParams {
	return sqlc.CreateUniversityParams{
		Name:                     req.Name,
		Slug:                     req.Slug,
		Overview:                 &req.Overview,
		Excerpt:                  &req.Excerpt,
		Country:                  &req.Country,
		State:                    &req.State,
		City:                     &req.City,
		FullLocation:             &req.FullLocation,
		CoverImage:               &req.CoverImage,
		Logo:                     &req.Logo,
		InstitutionType:          &req.InstitutionType,
		CampusSetting:            &req.CampusSetting,
		InStateTuition:           toPgNumeric(req.InStateTuition),
		OutOfStateTuition:        toPgNumeric(req.OutOfStateTuition),
		InternationalTuition:     toPgNumeric(req.InternationalTuition),
		NeedBasedAid:             req.NeedBasedAid,
		MeritScholarships:        req.MeritScholarships,
		WorkStudy:                req.WorkStudy,
		NoApplicationFee:         req.NoApplicationFee,
		AcceptanceRate:           toPgNumeric(req.AcceptanceRate),
		TestingPolicy:            &req.TestingPolicy,
		SatRange:                 &req.SatRange,
		ActRange:                 &req.ActRange,
		OnCampusHousing:          req.OnCampusHousing,
		FreshmenRequiredOnCampus: req.FreshmenRequiredOnCampus,
		ContactEmail:             &req.ContactEmail,
		ContactPhone:             &req.ContactPhone,
		Website:                  &req.Website,
	}
}

func toCreateUniversityResponse(u sqlc.University) *CreateUniversityResponse {
	return &CreateUniversityResponse{
		ID:                       u.ID,
		Name:                     u.Name,
		Slug:                     u.Slug,
		Overview:                 derefString(u.Overview),
		Excerpt:                  derefString(u.Excerpt),
		Country:                  derefString(u.Country),
		State:                    derefString(u.State),
		City:                     derefString(u.City),
		FullLocation:             derefString(u.FullLocation),
		CoverImage:               derefString(u.CoverImage),
		Logo:                     derefString(u.Logo),
		InstitutionType:          derefString(u.InstitutionType),
		CampusSetting:            derefString(u.CampusSetting),
		InStateTuition:           fromPgNumeric(u.InStateTuition),
		OutOfStateTuition:        fromPgNumeric(u.OutOfStateTuition),
		InternationalTuition:     fromPgNumeric(u.InternationalTuition),
		NeedBasedAid:             u.NeedBasedAid,
		MeritScholarships:        u.MeritScholarships,
		WorkStudy:                u.WorkStudy,
		NoApplicationFee:         u.NoApplicationFee,
		AcceptanceRate:           fromPgNumeric(u.AcceptanceRate),
		TestingPolicy:            derefString(u.TestingPolicy),
		SatRange:                 derefString(u.SatRange),
		ActRange:                 derefString(u.ActRange),
		OnCampusHousing:          u.OnCampusHousing,
		FreshmenRequiredOnCampus: u.FreshmenRequiredOnCampus,
		ContactEmail:             derefString(u.ContactEmail),
		ContactPhone:             derefString(u.ContactPhone),
		Website:                  derefString(u.Website),
		CreatedAt:                u.CreatedAt,
		UpdatedAt:                u.UpdatedAt,
	}
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func toPgNumeric(v float64) pgtype.Numeric {
	var n pgtype.Numeric
	_ = n.Scan(v)
	return n
}

func fromPgNumeric(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, _ := n.Float64Value()
	return f.Float64
}
