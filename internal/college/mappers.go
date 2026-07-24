package college

import (
	"time"

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
	}
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
		CoverImage:      deref(c.CoverImage),
		Logo:            deref(c.Logo),
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
		CreatedAt:       c.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:       c.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func toCollegeDetailResponse(c sqlc.College) *CollegeDetailResponse {
	return &CollegeDetailResponse{
		CreateCollegeResponse: *toCreateCollegeResponse(c),
	}
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
