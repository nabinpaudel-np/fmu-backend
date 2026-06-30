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
		Zipcode:                  &req.Zipcode,
		TuitionMin:               &req.TuitionMin,
		TuitionMax:               &req.TuitionMax,
		AvgHighSchoolGpa:         toPgNumeric(req.AvgHighSchoolGpa),
		FoundedYear:              toPgInt16(req.FoundedYear),
		CampusSize:               &req.CampusSize,
		GalleryImages:            req.GalleryImages,
		IsPopular:                req.IsPopular,
		IsFeatured:               req.IsFeatured,
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
		Zipcode:                  derefString(u.Zipcode),
		TuitionMin:               derefInt32(u.TuitionMin),
		TuitionMax:               derefInt32(u.TuitionMax),
		AvgHighSchoolGpa:         fromPgNumeric(u.AvgHighSchoolGpa),
		FoundedYear:              derefInt16AsInt32(u.FoundedYear),
		CampusSize:               derefString(u.CampusSize),
		GalleryImages:            u.GalleryImages,
		IsPopular:                u.IsPopular,
		IsFeatured:               u.IsFeatured,
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

func toUniversityListItem(u sqlc.University) UniversityListItem {
	return UniversityListItem{
		ID:              u.ID,
		Name:            u.Name,
		Slug:            u.Slug,
		Country:         derefString(u.Country),
		State:           derefString(u.State),
		City:            derefString(u.City),
		Logo:            derefString(u.Logo),
		CoverImage:      derefString(u.CoverImage),
		InstitutionType: derefString(u.InstitutionType),
		CampusSetting:   derefString(u.CampusSetting),
		TuitionMin:      derefInt32(u.TuitionMin),
		TuitionMax:      derefInt32(u.TuitionMax),
		AcceptanceRate:  fromPgNumeric(u.AcceptanceRate),
		IsPopular:       u.IsPopular,
		IsFeatured:      u.IsFeatured,
	}
}

func toUniversityDetailResponse(
	u sqlc.University,
	degreeLevels []sqlc.DegreeLevel,
	majors []sqlc.Major,
	studyFormats []sqlc.StudyFormat,
	specialAffiliations []sqlc.SpecialAffiliation,
	athletics []sqlc.Athletic,
	supportServices []sqlc.SupportService,
) *UniversityDetailResponse {
	return &UniversityDetailResponse{
		CreateUniversityResponse: *toCreateUniversityResponse(u),
		DegreeLevels:             toLookupItems[sqlc.DegreeLevel, DegreeLevelResponse](degreeLevels, func(d sqlc.DegreeLevel) DegreeLevelResponse { return DegreeLevelResponse{ID: d.ID, Name: d.Name} }),
		Majors:                   toLookupItems[sqlc.Major, MajorResponse](majors, func(m sqlc.Major) MajorResponse { return MajorResponse{ID: m.ID, Name: m.Name} }),
		StudyFormats:             toLookupItems[sqlc.StudyFormat, StudyFormatResponse](studyFormats, func(s sqlc.StudyFormat) StudyFormatResponse { return StudyFormatResponse{ID: s.ID, Name: s.Name} }),
		SpecialAffiliations:      toLookupItems[sqlc.SpecialAffiliation, SpecialAffiliationResponse](specialAffiliations, func(s sqlc.SpecialAffiliation) SpecialAffiliationResponse { return SpecialAffiliationResponse{ID: s.ID, Name: s.Name} }),
		Athletics:                toLookupItems[sqlc.Athletic, AthleticResponse](athletics, func(a sqlc.Athletic) AthleticResponse { return AthleticResponse{ID: a.ID, Name: a.Name} }),
		SupportServices:          toLookupItems[sqlc.SupportService, SupportServiceResponse](supportServices, func(s sqlc.SupportService) SupportServiceResponse { return SupportServiceResponse{ID: s.ID, Name: s.Name} }),
	}
}

func toLookupItems[In any, Out any](items []In, convert func(In) Out) []Out {
	out := make([]Out, len(items))
	for i, item := range items {
		out[i] = convert(item)
	}
	return out
}

func derefInt32(p *int32) int32 {
	if p == nil {
		return 0
	}
	return *p
}

func derefInt16AsInt32(p *int16) int32 {
	if p == nil {
		return 0
	}
	return int32(*p)
}

func toPgInt16(v int32) *int16 {
	if v == 0 {
		return nil
	}
	s := int16(v)
	return &s
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
