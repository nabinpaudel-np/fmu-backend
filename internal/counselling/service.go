package counselling

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"fmu-backend/internal/auth"
	"fmu-backend/internal/db/sqlc"
	"fmu-backend/internal/errs"
	"fmu-backend/internal/pagination"
)

// UniversityExister is the slim contract the counselling service needs from
// university — just enough to confirm a university exists before accepting
// an inquiry against it. Errors should wrap errs.ErrNotFound on miss.
type UniversityExister interface {
	GetByID(ctx context.Context, id string) error
}

// CollegeExister is the slim contract for college existence checks.
type CollegeExister interface {
	GetByID(ctx context.Context, id string) error
}

// CounsellingService is the public surface for the counselling package.
type CounsellingService interface {
	SubmitGeneral(ctx context.Context, req *SubmitGeneralRequest) (*SubmitResponse, error)
	SubmitUniversity(ctx context.Context, universityID string, req *SubmitSpecificRequest) (*SubmitResponse, error)
	SubmitCollege(ctx context.Context, collegeID string, req *SubmitSpecificRequest) (*SubmitResponse, error)
	GetByID(ctx context.Context, id string) (*CounsellingListItem, error)
	List(ctx context.Context, target TargetType, statusFilter string, q pagination.Query) ([]CounsellingListItem, int64, error)
	Update(ctx context.Context, id string, req *UpdateRequest) (*CounsellingListItem, error)
}

type counsellingService struct {
	repo       CounsellingRepository
	university UniversityExister
	college    CollegeExister
}

// NewCounsellingService wires the repository and the two existers. Callers
// with concrete service handles should use NewUniversityExisterAdapter /
// NewCollegeExisterAdapter to bridge them.
func NewCounsellingService(
	repo CounsellingRepository,
	university UniversityExister,
	college CollegeExister,
) CounsellingService {
	return &counsellingService{
		repo:       repo,
		university: university,
		college:    college,
	}
}

func (s *counsellingService) SubmitGeneral(ctx context.Context, req *SubmitGeneralRequest) (*SubmitResponse, error) {
	row, err := s.repo.Create(ctx, CreateParams{
		TargetType:          nil,
		TargetID:            pgtype.UUID{},
		FullName:            strings.TrimSpace(req.FullName),
		Email:               strings.TrimSpace(req.Email),
		Phone:               stringPtrOrNil(req.Phone),
		Country:             stringPtrOrNil(req.Country),
		PreferredUniversity: stringPtrOrNil(req.PreferredUniversity),
		ResumeURL:           stringPtrOrNil(req.ResumeURL),
	})
	if err != nil {
		log.Default().Printf("create general counselling inquiry: %v", err)
		return nil, err
	}
	return &SubmitResponse{
		InquiryID: row.ID,
		Type:      TargetGeneral,
		Status:    row.Status,
		CreatedAt: row.CreatedAt,
	}, nil
}

func (s *counsellingService) SubmitUniversity(ctx context.Context, universityID string, req *SubmitSpecificRequest) (*SubmitResponse, error) {
	if err := s.university.GetByID(ctx, universityID); err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, err
	}

	targetID, err := uuidFromString(universityID)
	if err != nil {
		return nil, errs.ErrNotFound
	}
	targetType := string(TargetUniversity)

	row, err := s.repo.Create(ctx, CreateParams{
		TargetType:        &targetType,
		TargetID:          targetID,
		FullName:          strings.TrimSpace(req.FullName),
		Email:             strings.TrimSpace(req.Email),
		Phone:             stringPtrOrNil(req.Phone),
		Country:           stringPtrOrNil(req.Country),
		ProgramOfInterest: stringPtrOrNil(req.ProgramOfInterest),
		StartTerm:         stringPtrOrNil(req.StartTerm),
		CurrentEducation:  stringPtrOrNil(req.CurrentEducation),
		TestScores:        stringPtrOrNil(req.TestScores),
		Message:           stringPtrOrNil(req.Message),
	})
	if err != nil {
		log.Default().Printf("create university counselling inquiry: %v", err)
		return nil, err
	}
	return submitSpecificResponse(row, TargetUniversity), nil
}

func (s *counsellingService) SubmitCollege(ctx context.Context, collegeID string, req *SubmitSpecificRequest) (*SubmitResponse, error) {
	if err := s.college.GetByID(ctx, collegeID); err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, err
	}

	targetID, err := uuidFromString(collegeID)
	if err != nil {
		return nil, errs.ErrNotFound
	}
	targetType := string(TargetCollege)

	row, err := s.repo.Create(ctx, CreateParams{
		TargetType:        &targetType,
		TargetID:          targetID,
		FullName:          strings.TrimSpace(req.FullName),
		Email:             strings.TrimSpace(req.Email),
		Phone:             stringPtrOrNil(req.Phone),
		Country:           stringPtrOrNil(req.Country),
		ProgramOfInterest: stringPtrOrNil(req.ProgramOfInterest),
		StartTerm:         stringPtrOrNil(req.StartTerm),
		CurrentEducation:  stringPtrOrNil(req.CurrentEducation),
		TestScores:        stringPtrOrNil(req.TestScores),
		Message:           stringPtrOrNil(req.Message),
	})
	if err != nil {
		log.Default().Printf("create college counselling inquiry: %v", err)
		return nil, err
	}
	return submitSpecificResponse(row, TargetCollege), nil
}

func submitSpecificResponse(row sqlc.CounsellingInquiry, t TargetType) *SubmitResponse {
	targetID := uuidStringPtr(row.TargetID)
	return &SubmitResponse{
		InquiryID: row.ID,
		Type:      t,
		TargetID:  targetID,
		Status:    row.Status,
		CreatedAt: row.CreatedAt,
	}
}

func (s *counsellingService) GetByID(ctx context.Context, id string) (*CounsellingListItem, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
			return nil, errs.ErrNotFound
		}
		log.Default().Printf("get counselling inquiry %s: %v", id, err)
		return nil, err
	}
	if err := s.assertReadScope(ctx, row); err != nil {
		return nil, err
	}
	item := toListItem(fromGetRow(row))
	return &item, nil
}

func (s *counsellingService) List(ctx context.Context, target TargetType, statusFilter string, q pagination.Query) ([]CounsellingListItem, int64, error) {
	if statusFilter != "" && statusFilter != StatusPending && statusFilter != StatusReviewed && statusFilter != StatusArchived {
		return nil, 0, errs.ErrBadRequest
	}
	if target != "" && !target.IsValid() {
		return nil, 0, errs.ErrBadRequest
	}

	// Resolve the caller's scope. Admins have no scope (see everything).
	// Representatives are pinned to their institution. Other roles are 403.
	scopeType, scopeID, err := s.readScopeFilter(ctx)
	if err != nil {
		return nil, 0, err
	}

	params := ListParams{
		Status:     statusFilter,
		TargetType: scopeType,
		TargetID:   scopeID,
		Limit:      int32(q.Limit()),
		Offset:     int32(q.Offset()),
	}

	// Caller asked for a specific target filter — intersect with scope.
	// Representative scope takes precedence; explicit ?type= is only
	// honored when it doesn't widen beyond what the rep is allowed to see.
	switch target {
	case TargetGeneral:
		// Reps cannot see general rows (no target).
		if scopeType != "" {
			return []CounsellingListItem{}, 0, nil
		}
		params.TargetType = generalSentinel
	case TargetUniversity, TargetCollege:
		// Caller narrowed to a specific institution type.
		if scopeType != "" && scopeType != string(target) {
			return []CounsellingListItem{}, 0, nil
		}
		params.TargetType = string(target)
	}

	total, err := s.repo.Count(ctx, params)
	if err != nil {
		log.Default().Printf("count counselling inquiries: %v", err)
		return nil, 0, err
	}
	if total == 0 {
		return []CounsellingListItem{}, 0, nil
	}

	rows, err := s.repo.List(ctx, params)
	if err != nil {
		log.Default().Printf("list counselling inquiries: %v", err)
		return nil, 0, err
	}
	items := make([]CounsellingListItem, len(rows))
	for i, r := range rows {
		items[i] = toListItem(fromListRow(r))
	}
	return items, total, nil
}

// generalSentinel is passed as target_type to the SQL when the caller wants
// only general (target_type IS NULL) rows. The SQL branches on this value
// (see internal/db/queries/counselling.sql).
const generalSentinel = "__general__"

// readScopeFilter returns the (target_type, target_id) filter a caller is
// allowed to use. Admins get ("", "") meaning "no scope restriction".
// Representatives get (their type, their institution id). Students are
// rejected with 403 — they have no business in the counselling dashboard.
func (s *counsellingService) readScopeFilter(ctx context.Context) (targetType, targetID string, err error) {
	claims, err := auth.ClaimsFromContext(ctx)
	if err != nil {
		return "", "", errs.ErrUnauthorized
	}
	switch claims.Role {
	case auth.RoleAdmin:
		return "", "", nil
	case auth.RoleRepresentative:
		if claims.RepresentativeUniversityID != "" {
			return string(TargetUniversity), claims.RepresentativeUniversityID, nil
		}
		if claims.RepresentativeCollegeID != "" {
			return string(TargetCollege), claims.RepresentativeCollegeID, nil
		}
		return "", "", errs.ErrForbidden
	default:
		return "", "", errs.ErrForbidden
	}
}

// assertReadScope gates GetByID. Admins see anything; reps only see rows
// attached to their own institution; general rows are admin-only.
func (s *counsellingService) assertReadScope(ctx context.Context, row sqlc.GetCounsellingInquiryByIDRow) error {
	claims, err := auth.ClaimsFromContext(ctx)
	if err != nil {
		return nil // anonymous caller — auth middleware should reject before here
	}
	if claims.Role == auth.RoleAdmin {
		return nil
	}
	if claims.Role != auth.RoleRepresentative {
		return errs.ErrForbidden
	}
	if row.TargetType == nil {
		return errs.ErrForbidden
	}
	targetID := uuidStringPtr(row.TargetID)
	if targetID == nil {
		return errs.ErrForbidden
	}
	switch TargetType(*row.TargetType) {
	case TargetUniversity:
		if claims.RepresentativeUniversityID != "" && *targetID == claims.RepresentativeUniversityID {
			return nil
		}
	case TargetCollege:
		if claims.RepresentativeCollegeID != "" && *targetID == claims.RepresentativeCollegeID {
			return nil
		}
	}
	return errs.ErrForbidden
}

func (s *counsellingService) assertUpdateScope(ctx context.Context, row sqlc.GetCounsellingInquiryByIDRow) error {
	return s.assertReadScope(ctx, row)
}

func (s *counsellingService) Update(ctx context.Context, id string, req *UpdateRequest) (*CounsellingListItem, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrNotFound
		}
		log.Default().Printf("get counselling inquiry %s for update: %v", id, err)
		return nil, err
	}

	if err := s.assertUpdateScope(ctx, row); err != nil {
		return nil, err
	}

	claims, _ := auth.ClaimsFromContext(ctx)
	reviewerID := pgtype.UUID{}
	if claims != nil && claims.UserID != "" {
		if u, err := uuidFromString(claims.UserID); err == nil {
			reviewerID = u
		}
	}

	reviewedAt := pgtype.Timestamptz{Valid: false}
	if req.Status == StatusReviewed {
		reviewedAt = pgtype.Timestamptz{Time: nowUTC(), Valid: true}
	}

	if _, err := s.repo.Update(ctx, UpdateParams{
		ID:         id,
		Status:     req.Status,
		ReviewerID: reviewerID,
		ReviewedAt: reviewedAt,
		ReviewNote: stringPtrOrNil(req.ReviewNote),
	}); err != nil {
		log.Default().Printf("update counselling inquiry %s: %v", id, err)
		return nil, err
	}

	fresh, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.Default().Printf("refetch counselling inquiry %s: %v", id, err)
		return nil, err
	}
	item := toListItem(fromGetRow(fresh))
	return &item, nil
}

func nowUTC() time.Time { return time.Now().UTC() }
