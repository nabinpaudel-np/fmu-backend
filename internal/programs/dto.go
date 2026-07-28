package programs

type CreateProgramRequest struct {
	Title       string `json:"title"       validate:"required,min=1,max=255"`
	Description string `json:"description" validate:"required"`
	DegreeID    string `json:"degree_id"   validate:"required,uuid"`
}

type ProgramResponse struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	DegreeID    string `json:"degree_id"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// ProgramLookupItem is the slim shape used in the bundled /lookups endpoint so
// the frontend can populate form dropdowns without the full description body.
type ProgramLookupItem struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	DegreeID string `json:"degree_id"`
}
