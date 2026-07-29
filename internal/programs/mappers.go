package programs

import (
	"time"

	"fmu-backend/internal/db/sqlc"
)

func toCreateParams(req *CreateProgramRequest) sqlc.CreateProgramParams {
	return sqlc.CreateProgramParams{
		Title:         req.Title,
		Description:   req.Description,
		Excerpt:       ptr(req.Excerpt),
		CareerOptions: ptr(req.CareerOptions),
		DegreeID:      req.DegreeID,
	}
}

func toUpdateParams(id string, req *UpdateProgramRequest) sqlc.UpdateProgramParams {
	return sqlc.UpdateProgramParams{
		ID:            id,
		Title:         req.Title,
		Description:   req.Description,
		Excerpt:       ptr(req.Excerpt),
		CareerOptions: ptr(req.CareerOptions),
		DegreeID:      req.DegreeID,
	}
}

func toResponse(p sqlc.Program) *ProgramResponse {
	return &ProgramResponse{
		ID:            p.ID,
		Title:         p.Title,
		Description:   p.Description,
		Excerpt:       p.Excerpt,
		CareerOptions: p.CareerOptions,
		DegreeID:      p.DegreeID,
		CreatedAt:     p.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:     p.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func toLookupItem(p sqlc.Program) ProgramLookupItem {
	return ProgramLookupItem{
		ID:       p.ID,
		Title:    p.Title,
		DegreeID: p.DegreeID,
	}
}

func ptr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
