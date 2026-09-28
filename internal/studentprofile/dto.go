package studentprofile

import "time"

// UpdateProfileRequest is the body for PUT /api/v1/student/profile. All
// fields are optional — pointer types let the caller distinguish "omit"
// (no change) from "send the zero value" (explicit update). At least one
// field must be present; sending an empty body is rejected by the service
// layer so the endpoint can never be a no-op.
//
//   - Budget is the student's max spending power. Matched against
//     university.tuition_min (permissive rule — see /recommendations).
//   - IntendedCountry is matched against university.country exactly. The
//     value must be one of the country strings already used in the
//     universities table (callers can list distinct values via the
//     /universities endpoint).
//   - CurrentEducation is stored for the student's record. Allowed
//     values: "8-10", "10-12", "bachelors", "masters", "phd".
//   - ProgramIDs is a pointer to a slice so the caller can distinguish
//     "omit (keep existing programs)" from "send [] (clear all)". Matched
//     via the programs' degree_id → university_degree_levels overlap.
type UpdateProfileRequest struct {
	Budget           *int64    `json:"budget,omitempty"            validate:"omitempty,min=0"            example:"50000"`
	IntendedCountry  *string   `json:"intended_country,omitempty"   validate:"omitempty,max=100"          example:"USA"`
	CurrentEducation *string   `json:"current_education,omitempty"  validate:"omitempty,oneof=8-10 10-12 bachelors masters phd" example:"bachelors"`
	ProgramIDs       *[]string `json:"program_ids,omitempty"        validate:"omitempty,max=20,dive,uuid" example:"[\"d3b07384-d9a2-4e0a-b71e-1c9f3e3e0a1b\"]"`
}

// ProfileResponse mirrors the saved profile. ProgramIDs is always a
// (possibly empty) slice so the frontend can `v-for` over it without
// nil-checks.
type ProfileResponse struct {
	UserID           string    `json:"user_id"                                       example:"d3b07384-d9a2-4e0a-b71e-1c9f3e3e0a1b"`
	Budget           *int64    `json:"budget,omitempty"                              example:"50000"`
	IntendedCountry  *string   `json:"intended_country,omitempty"                    example:"USA"`
	CurrentEducation *string   `json:"current_education,omitempty"                   example:"bachelors"`
	ProgramIDs       []string  `json:"program_ids"                                  example:"[\"d3b07384-d9a2-4e0a-b71e-1c9f3e3e0a1b\"]"`
	CreatedAt        time.Time `json:"created_at"                                    example:"2026-09-21T10:00:00Z"`
	UpdatedAt        time.Time `json:"updated_at"                                    example:"2026-09-21T10:00:00Z"`
}