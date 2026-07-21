package favorites

import (
	"fmu-backend/internal/college"
	"fmu-backend/internal/university"
)

// Re-export the existing list-item types under the favorites URL namespace.
// One-way import: favorites depends on university/college, never the reverse.
type (
	UniversityListItem = university.UniversityListItem
	CollegeListItem    = college.CollegeListItem
)