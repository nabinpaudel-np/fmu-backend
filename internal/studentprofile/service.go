package studentprofile

import (
	"context"
	"fmt"
	"log"

	"fmu-backend/internal/db/sqlc"
	"fmu-backend/internal/errs"
	"fmu-backend/internal/pagination"
	"fmu-backend/internal/university"
)

// favoritesLookup is the minimal favorites dependency the recommendation
// list needs to stamp `is_favorited`. favorites.Repository satisfies it
// implicitly — defined here so this package only imports what it uses.
type favoritesLookup interface {
	FavoritedUniversityIDs(ctx context.Context, userID string, ids []string) (map[string]struct{}, error)
}

// universitiesLookup is the minimal university dependency the
// recommendation list needs to stamp `has_representative`.
// university.UniversityService satisfies it implicitly.
type universitiesLookup interface {
	RepresentedIDs(ctx context.Context, ids []string) (map[string]struct{}, error)
}

// Service exposes the student profile lifecycle and the recommendations
// derived from it. All write paths validate that referenced programs
// exist (returning ErrInvalidProgramRefs on a dangling ID) and that at
// least one field is supplied for upserts.
type Service interface {
	Upsert(ctx context.Context, userID string, req *UpdateProfileRequest) (*ProfileResponse, error)
	Get(ctx context.Context, userID string) (*ProfileResponse, error)
	Delete(ctx context.Context, userID string) error
	ListRecommendations(ctx context.Context, userID string, q pagination.Query) ([]university.UniversityListItem, int64, error)
}

// ErrInvalidProgramRefs is returned by Upsert when one or more program
// IDs in the request don't exist in the programs table. The handler
// converts this to a 400 with a field-level error detail.
type InvalidProgramRefsError struct {
	Missing []string
}

func (e *InvalidProgramRefsError) Error() string {
	return fmt.Sprintf("invalid program_ids: %v", e.Missing)
}

type profileService struct {
	repo        Repository
	favorites   favoritesLookup
	universities universitiesLookup
}

func NewService(repo Repository, favs favoritesLookup, unis universitiesLookup) Service {
	return &profileService{repo: repo, favorites: favs, universities: unis}
}

func (s *profileService) Upsert(ctx context.Context, userID string, req *UpdateProfileRequest) (*ProfileResponse, error) {
	if !hasAnyField(req) {
		return nil, errs.ErrBadRequest
	}

	// Validate program references only when the caller is actually sending
	// programs (non-nil). A nil pointer means "leave existing programs alone"
	// — matching the COALESCE-style preservation we do for the scalar fields.
	if req.ProgramIDs != nil && len(*req.ProgramIDs) > 0 {
		existing, err := s.repo.ExistingProgramIDs(ctx, *req.ProgramIDs)
		if err != nil {
			log.Default().Printf("student profile upsert user=%s: validate programs: %v", userID, err)
			return nil, err
		}
		var missing []string
		for _, id := range *req.ProgramIDs {
			if _, ok := existing[id]; !ok {
				missing = append(missing, id)
			}
		}
		if len(missing) > 0 {
			return nil, &InvalidProgramRefsError{Missing: missing}
		}
	}

	row, err := s.repo.Upsert(ctx, userID, sqlc.StudentProfile{
		UserID:           userID,
		Budget:           req.Budget,
		IntendedCountry:  req.IntendedCountry,
		CurrentEducation: req.CurrentEducation,
	})
	if err != nil {
		log.Default().Printf("student profile upsert user=%s: %v", userID, err)
		return nil, err
	}

	// Replace program ids only when the caller sent the field. nil = leave
	// existing programs alone; non-nil (even []) = replace.
	if req.ProgramIDs != nil {
		if err := s.repo.ReplaceProgramIDs(ctx, userID, *req.ProgramIDs); err != nil {
			log.Default().Printf("student profile upsert user=%s replace programs: %v", userID, err)
			return nil, err
		}
	}

	// Build response with the post-write program list. If the caller didn't
	// touch programs, fetch the current ones so the response stays accurate.
	var programIDs []string
	if req.ProgramIDs != nil {
		programIDs = *req.ProgramIDs
	} else {
		programIDs, err = s.repo.ListProgramIDs(ctx, userID)
		if err != nil {
			log.Default().Printf("student profile upsert user=%s list programs: %v", userID, err)
			return nil, err
		}
	}

	return toResponse(row, programIDs), nil
}

func (s *profileService) Get(ctx context.Context, userID string) (*ProfileResponse, error) {
	row, err := s.repo.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	programIDs, err := s.repo.ListProgramIDs(ctx, userID)
	if err != nil {
		log.Default().Printf("student profile get user=%s list programs: %v", userID, err)
		return nil, err
	}
	return toResponse(row, programIDs), nil
}

func (s *profileService) Delete(ctx context.Context, userID string) error {
	if err := s.repo.Delete(ctx, userID); err != nil {
		log.Default().Printf("student profile delete user=%s: %v", userID, err)
		return err
	}
	return nil
}

// ListRecommendations derives a list from the saved profile. The profile
// must exist with at least one of {budget, country, programs} set —
// current_education is storage-only and doesn't count. Empty profiles
// are a 400 so the caller is nudged to fill the form before polling.
func (s *profileService) ListRecommendations(ctx context.Context, userID string, q pagination.Query) ([]university.UniversityListItem, int64, error) {
	profile, err := s.repo.Get(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	programIDs, err := s.repo.ListProgramIDs(ctx, userID)
	if err != nil {
		log.Default().Printf("student profile list recs user=%s list programs: %v", userID, err)
		return nil, 0, err
	}
	if !profileHasAnyField(profile, programIDs) {
		return nil, 0, errs.ErrBadRequest
	}

	items, total, err := s.repo.ListRecommendedUniversities(ctx, profile, programIDs, q)
	if err != nil {
		log.Default().Printf("student profile list recs user=%s: %v", userID, err)
		return nil, 0, err
	}

	s.stampFavorited(ctx, userID, items)
	if err := s.stampRepresented(ctx, items); err != nil {
		log.Default().Printf("student profile stamp represented: %v", err)
		return nil, 0, err
	}
	return items, total, nil
}

// stampFavorited mirrors internal/university/handler.go:54 — sets
// IsFavorited=true for any item in the user's favorites set. Silently
// no-ops on error (matches the existing handler's behavior).
func (s *profileService) stampFavorited(ctx context.Context, userID string, items []university.UniversityListItem) {
	if len(items) == 0 {
		return
	}
	ids := make([]string, len(items))
	for i, it := range items {
		ids[i] = it.ID
	}
	set, err := s.favorites.FavoritedUniversityIDs(ctx, userID, ids)
	if err != nil {
		return
	}
	for i := range items {
		if _, ok := set[items[i].ID]; ok {
			items[i].IsFavorited = true
		}
	}
}

func (s *profileService) stampRepresented(ctx context.Context, items []university.UniversityListItem) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]string, len(items))
	for i, it := range items {
		ids[i] = it.ID
	}
	set, err := s.universities.RepresentedIDs(ctx, ids)
	if err != nil {
		return err
	}
	for i := range items {
		_, items[i].HasRepresentative = set[items[i].ID]
	}
	return nil
}

// hasAnyField returns true if at least one updatable field is non-nil
// and non-zero. Used to reject empty PUT bodies.
func hasAnyField(req *UpdateProfileRequest) bool {
	if req.Budget != nil {
		return true
	}
	if req.IntendedCountry != nil {
		return true
	}
	if req.CurrentEducation != nil {
		return true
	}
	if req.ProgramIDs != nil {
		return true
	}
	return false
}

// profileHasAnyField checks the persisted profile. current_education is
// storage-only, so it doesn't count toward "is this profile useful for
// recommendations?" — same rule as the input-side check. Program IDs
// live in a separate junction table so they're passed in.
func profileHasAnyField(p sqlc.StudentProfile, programIDs []string) bool {
	if p.Budget != nil {
		return true
	}
	if p.IntendedCountry != nil && *p.IntendedCountry != "" {
		return true
	}
	if len(programIDs) > 0 {
		return true
	}
	return false
}

func toResponse(row sqlc.StudentProfile, programIDs []string) *ProfileResponse {
	if programIDs == nil {
		programIDs = []string{}
	}
	return &ProfileResponse{
		UserID:           row.UserID,
		Budget:           row.Budget,
		IntendedCountry:  row.IntendedCountry,
		CurrentEducation: row.CurrentEducation,
		ProgramIDs:       programIDs,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}