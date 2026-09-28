package favorites

import (
	"fmu-backend/internal/college"
	"fmu-backend/internal/scholarship"
	"fmu-backend/internal/university"
)

// Re-export the existing list-item types under the favorites URL namespace.
// One-way import: favorites depends on university/college/scholarship, never
// the reverse.
type (
	UniversityListItem  = university.UniversityListItem
	CollegeListItem     = college.CollegeListItem
	ScholarshipListItem = scholarship.ScholarshipListItem
)
