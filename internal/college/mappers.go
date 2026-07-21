package college

import (
	"time"

	"fmu-backend/internal/db/sqlc"
)

func toCreateCollegeParams(req *CreateCollegeRequest) sqlc.CreateCollegeParams {
	return sqlc.CreateCollegeParams{
		Name:         req.Name,
		Slug:         req.Slug,
		UniversityID: req.UniversityID,
		Overview:     req.Overview,
		Country:      stringPtrOrNil(req.Country),
		State:        stringPtrOrNil(req.State),
		City:         stringPtrOrNil(req.City),
		FullLocation: stringPtrOrNil(req.FullLocation),
		Logo:         stringPtrOrNil(req.Logo),
	}
}

func toCreateCollegeResponse(c sqlc.College) *CreateCollegeResponse {
	return &CreateCollegeResponse{
		ID:           c.ID,
		Name:         c.Name,
		Slug:         c.Slug,
		UniversityID: c.UniversityID,
		Overview:     c.Overview,
		Country:      deref(c.Country),
		State:        deref(c.State),
		City:         deref(c.City),
		FullLocation: deref(c.FullLocation),
		Logo:         deref(c.Logo),
		CreatedAt:    c.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:    c.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func toCollegeDetailResponse(c sqlc.College) *CollegeDetailResponse {
	return &CollegeDetailResponse{
		CreateCollegeResponse: *toCreateCollegeResponse(c),
	}
}

func toCollegeListItem(c sqlc.College) CollegeListItem {
	return CollegeListItem{
		ID:           c.ID,
		Name:         c.Name,
		Slug:         c.Slug,
		UniversityID: c.UniversityID,
		Country:      deref(c.Country),
		State:        deref(c.State),
		City:         deref(c.City),
		Logo:         deref(c.Logo),
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
		Country:    r.Country,
		State:      r.State,
		City:       r.City,
		Logo:       r.Logo,
	}
}

func stringPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
