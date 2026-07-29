package college

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"fmu-backend/internal/db/sqlc"
)

func toCreateCollegeParams(req *CreateCollegeRequest) sqlc.CreateCollegeParams {
	return sqlc.CreateCollegeParams{
		Name:            req.Name,
		Slug:            req.Slug,
		UniversityID:    req.UniversityID,
		Overview:        req.Overview,
		Excerpt:         stringPtrOrNil(req.Excerpt),
		Country:         stringPtrOrNil(req.Country),
		State:           stringPtrOrNil(req.State),
		City:            stringPtrOrNil(req.City),
		FullLocation:    stringPtrOrNil(req.FullLocation),
		CoverImage:      stringPtrOrNil(req.CoverImage),
		Logo:            stringPtrOrNil(req.Logo),
		InstitutionType: stringPtrOrNil(req.InstitutionType),
		CampusSetting:   stringPtrOrNil(req.CampusSetting),
		ContactEmail:    stringPtrOrNil(req.ContactEmail),
		ContactPhone:    stringPtrOrNil(req.ContactPhone),
		Website:         stringPtrOrNil(req.Website),
		Zipcode:         stringPtrOrNil(req.Zipcode),
		FoundedYear:     int16PtrOrNil(req.FoundedYear),
		CampusSize:      stringPtrOrNil(req.CampusSize),
		GalleryImages:   req.GalleryImages,
		IsPopular:       req.IsPopular,
		IsFeatured:      req.IsFeatured,
		FullAddress:     stringPtrOrNil(req.FullAddress),
		MapsUrl:         stringPtrOrNil(req.MapsUrl),
		SeoTitle:        stringPtrOrNil(req.SeoTitle),
		SeoDescription:  stringPtrOrNil(req.SeoDescription),
		Status:          statusOrDefault(req.Status),
	}
}

func statusOrDefault(s string) string {
	if s == "" {
		return "published"
	}
	return s
}

func toCreateCollegeResponse(c sqlc.College) *CreateCollegeResponse {
	return &CreateCollegeResponse{
		ID:              c.ID,
		Name:            c.Name,
		Slug:            c.Slug,
		UniversityID:    c.UniversityID,
		Overview:        c.Overview,
		Excerpt:         deref(c.Excerpt),
		Country:         deref(c.Country),
		State:           deref(c.State),
		City:            deref(c.City),
		FullLocation:    deref(c.FullLocation),
		Zipcode:         deref(c.Zipcode),
		FullAddress:     deref(c.FullAddress),
		CoverImage:      deref(c.CoverImage),
		Logo:            deref(c.Logo),
		MapsUrl:         deref(c.MapsUrl),
		GalleryImages:   c.GalleryImages,
		InstitutionType: deref(c.InstitutionType),
		CampusSetting:   deref(c.CampusSetting),
		ContactEmail:    deref(c.ContactEmail),
		ContactPhone:    deref(c.ContactPhone),
		Website:         deref(c.Website),
		FoundedYear:     derefInt16(c.FoundedYear),
		CampusSize:      deref(c.CampusSize),
		IsPopular:       c.IsPopular,
		IsFeatured:      c.IsFeatured,
		SeoTitle:        deref(c.SeoTitle),
		SeoDescription:  deref(c.SeoDescription),
		Status:          c.Status,
		PublishedAt:     publishedAtPtr(c.PublishedAt),
		CreatedAt:       c.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:       c.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func publishedAtPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

func toCollegeDetailResponse(
	c sqlc.College,
	degreeLevels []sqlc.DegreeLevel,
	majors []sqlc.Major,
	studyFormats []sqlc.StudyFormat,
) *CollegeDetailResponse {
	return &CollegeDetailResponse{
		CreateCollegeResponse: *toCreateCollegeResponse(c),
		DegreeLevels: toLookupItems[sqlc.DegreeLevel, CollegeDegreeLevelResponse](degreeLevels, func(d sqlc.DegreeLevel) CollegeDegreeLevelResponse {
			return CollegeDegreeLevelResponse{ID: d.ID, Name: d.Name}
		}),
		Majors: toLookupItems[sqlc.Major, CollegeMajorResponse](majors, func(m sqlc.Major) CollegeMajorResponse {
			return CollegeMajorResponse{ID: m.ID, Name: m.Name}
		}),
		StudyFormats: toLookupItems[sqlc.StudyFormat, CollegeStudyFormatResponse](studyFormats, func(s sqlc.StudyFormat) CollegeStudyFormatResponse {
			return CollegeStudyFormatResponse{ID: s.ID, Name: s.Name}
		}),
	}
}

func toLookupItems[In any, Out any](items []In, convert func(In) Out) []Out {
	out := make([]Out, len(items))
	for i, item := range items {
		out[i] = convert(item)
	}
	return out
}

func toCollegeListItem(c sqlc.College) CollegeListItem {
	return CollegeListItem{
		ID:              c.ID,
		Name:            c.Name,
		Slug:            c.Slug,
		UniversityID:    c.UniversityID,
		Country:         deref(c.Country),
		State:           deref(c.State),
		City:            deref(c.City),
		Logo:            deref(c.Logo),
		CoverImage:      deref(c.CoverImage),
		InstitutionType: deref(c.InstitutionType),
		CampusSetting:   deref(c.CampusSetting),
		IsPopular:       c.IsPopular,
		IsFeatured:      c.IsFeatured,
	}
}

func toCollegeSearchResult(r sqlc.SearchCollegesRow) CollegeSearchResult {
	return CollegeSearchResult{
		ID:   r.ID,
		Name: r.Name,
		Slug: r.Slug,
		University: CollegeUniversitySummary{
			ID:   r.UniversityID,
			Name: r.UniversityName,
			Slug: r.UniversitySlug,
			Logo: r.UniversityLogo,
		},
		Country: r.Country,
		State:   r.State,
		City:    r.City,
		Logo:    r.Logo,
	}
}

func stringPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func int16PtrOrNil(v int32) *int16 {
	if v == 0 {
		return nil
	}
	value := int16(v)
	return &value
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func derefInt16(p *int16) int32 {
	if p == nil {
		return 0
	}
	return int32(*p)
}
