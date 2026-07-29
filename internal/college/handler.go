package college

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"fmu-backend/internal/auth"
	"fmu-backend/internal/errs"
	"fmu-backend/internal/pagination"
	"fmu-backend/internal/response"
	"fmu-backend/internal/validator"
)

// collegeResourceField maps a lookup-table name to the request-DTO field
// that references it, so error messages name the field the client sent.
// Mirrors university.resourceField but only covers the three lookups
// colleges actually use.
var collegeResourceField = map[string]string{
	"degree_levels": "degree_level_ids",
	"majors":        "major_ids",
	"study_formats": "study_format_ids",
}

// formatMissingCollegeIDs caps the list at 10 IDs so a payload with hundreds
// of bad IDs doesn't bloat the error response.
func formatMissingCollegeIDs(ids []string) string {
	const cap = 10
	if len(ids) <= cap {
		return strings.Join(ids, ", ")
	}
	return strings.Join(ids[:cap], ", ") + fmt.Sprintf(" (and %d more)", len(ids)-cap)
}

// favoritesLookup is the minimal favorites dependency the public list/search
// handlers need to stamp `is_favorited`. Defined inline so this package only
// imports what it uses — favorites.Repository satisfies it implicitly.
type favoritesLookup interface {
	FavoritedCollegeIDs(ctx context.Context, userID string, ids []string) (map[string]struct{}, error)
}

type CollegeHandler struct {
	collegeService CollegeService
	favorites      favoritesLookup
}

func NewCollegeHandler(collegeService CollegeService, favs favoritesLookup) *CollegeHandler {
	return &CollegeHandler{
		collegeService: collegeService,
		favorites:      favs,
	}
}

// stampFavorited sets IsFavorited on each item in-place. No-op for anonymous
// requests (OptionalAuthMiddleware didn't inject claims) and on lookup errors.
func (h *CollegeHandler) stampFavorited(ctx context.Context, items []CollegeListItem) {
	uid, ok := auth.OptionalUserID(ctx)
	if !ok || len(items) == 0 {
		return
	}
	ids := make([]string, len(items))
	for i, it := range items {
		ids[i] = it.ID
	}
	set, err := h.favorites.FavoritedCollegeIDs(ctx, uid, ids)
	if err != nil {
		return
	}
	for i := range items {
		if _, ok := set[items[i].ID]; ok {
			items[i].IsFavorited = true
		}
	}
}

func (h *CollegeHandler) stampRepresented(ctx context.Context, items []CollegeListItem) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]string, len(items))
	for i, item := range items {
		ids[i] = item.ID
	}
	set, err := h.collegeService.RepresentedIDs(ctx, ids)
	if err != nil {
		return err
	}
	for i := range items {
		_, items[i].HasRepresentative = set[items[i].ID]
	}
	return nil
}

func (h *CollegeHandler) stampSearchRepresented(ctx context.Context, items []CollegeSearchResult) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]string, len(items))
	for i, item := range items {
		ids[i] = item.ID
	}
	set, err := h.collegeService.RepresentedIDs(ctx, ids)
	if err != nil {
		return err
	}
	for i := range items {
		_, items[i].HasRepresentative = set[items[i].ID]
	}
	return nil
}

func (h *CollegeHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateCollegeRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Drafts skip the full required-field validation — only name + slug
	// are required for drafts (enforced in the service). The DTO carries
	// `validate:"required"` on the rest, so we have to bypass the strict
	// pass when the caller is explicitly asking for a draft.
	if req.Status != "draft" {
		if err := validator.Validate.Struct(&req); err != nil {
			fields := validator.GetValidationErrors(err)
			response.ValidationError(w, http.StatusBadRequest, fields)
			return
		}
	}

	res, err := h.collegeService.Create(r.Context(), &req)
	if err != nil {
		var pubErr *errs.PublishValidationError
		if errors.As(err, &pubErr) {
			details := make([]response.ErrorDetail, 0, len(pubErr.Fields))
			for _, f := range pubErr.Fields {
				details = append(details, response.ErrorDetail{
					Field:   f,
					Message: "required",
				})
			}
			response.ValidationError(w, http.StatusBadRequest, details)
			return
		}
		switch {
		case errors.Is(err, errs.ErrCollegeSlugTaken):
			response.Error(w, http.StatusConflict, err.Error())
		case errors.Is(err, errs.ErrCollegeUniversityNotFound):
			response.Error(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, errs.ErrRepOutOfScope):
			response.Error(w, http.StatusForbidden, err.Error())
		default:
			response.Error(w, http.StatusInternalServerError, "something went wrong")
		}
		return
	}
	response.Success(w, http.StatusCreated, res)
}

func (h *CollegeHandler) Publish(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	res, err := h.collegeService.Publish(r.Context(), id)
	if err != nil {
		var pubErr *errs.PublishValidationError
		if errors.As(err, &pubErr) {
			details := make([]response.ErrorDetail, 0, len(pubErr.Fields))
			for _, f := range pubErr.Fields {
				details = append(details, response.ErrorDetail{
					Field:   f,
					Message: "required to publish",
				})
			}
			response.ValidationError(w, http.StatusBadRequest, details)
			return
		}
		if errors.Is(err, errs.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "college not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	response.Success(w, http.StatusOK, res)
}

func (h *CollegeHandler) List(w http.ResponseWriter, r *http.Request) {
	q := pagination.Parse(r)
	filters := ParseFilters(r.URL.Query())

	// Non-admin callers can only see published rows. We silently narrow the
	// status filter to "published" so an attempted `?status=draft` from a
	// public client returns the public list rather than a 403.
	if !isAdmin(r.Context()) {
		filters.Status = "published"
	}

	items, total, err := h.collegeService.List(r.Context(), q, filters)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	if err := h.stampRepresented(r.Context(), items); err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	h.stampFavorited(r.Context(), items)

	response.Success(w, http.StatusOK, pagination.Response[CollegeListItem]{
		Items: items,
		Meta:  q.BuildMeta(total),
	})
}

func (h *CollegeHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	detail, err := h.collegeService.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "college not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	// Non-admin callers see only published colleges. Drafts and
	// archived rows are 404'd so they don't leak existence.
	if !isAdmin(r.Context()) && detail.Status != "published" {
		response.Error(w, http.StatusNotFound, "college not found")
		return
	}

	set, err := h.collegeService.RepresentedIDs(r.Context(), []string{detail.ID})
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	_, detail.HasRepresentative = set[detail.ID]

	response.Success(w, http.StatusOK, detail)
}

func (h *CollegeHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req UpdateCollegeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validator.Validate.Struct(&req); err != nil {
		fields := validator.GetValidationErrors(err)
		response.ValidationError(w, http.StatusBadRequest, fields)
		return
	}

	res, err := h.collegeService.Update(r.Context(), id, &req)
	if err != nil {
		var refErr *errs.InvalidReferencesError
		if errors.As(err, &refErr) {
			details := make([]response.ErrorDetail, 0, len(refErr.References))
			for resource, ids := range refErr.References {
				details = append(details, response.ErrorDetail{
					Field:   collegeResourceField[resource],
					Message: fmt.Sprintf("the following %s do not exist: [%s]", resource, formatMissingCollegeIDs(ids)),
				})
			}
			response.ValidationError(w, http.StatusBadRequest, details)
			return
		}
		switch {
		case errors.Is(err, errs.ErrNotFound):
			response.Error(w, http.StatusNotFound, "college not found")
		case errors.Is(err, errs.ErrRepCannotChangeNameOrSlug):
			response.Error(w, http.StatusForbidden, err.Error())
		case errors.Is(err, errs.ErrCollegeSlugTaken):
			response.Error(w, http.StatusConflict, err.Error())
		default:
			response.Error(w, http.StatusInternalServerError, "something went wrong")
		}
		return
	}
	response.Success(w, http.StatusOK, res)
}

func (h *CollegeHandler) ListByUniversity(w http.ResponseWriter, r *http.Request) {
	universityID := chi.URLParam(r, "universityID")
	q := pagination.Parse(r)

	items, total, err := h.collegeService.ListByUniversity(r.Context(), universityID, q)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "university not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	if err := h.stampRepresented(r.Context(), items); err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	h.stampFavorited(r.Context(), items)

	response.Success(w, http.StatusOK, pagination.Response[CollegeListItem]{
		Items: items,
		Meta:  q.BuildMeta(total),
	})
}

func (h *CollegeHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		response.Error(w, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}
	if len(q) > 200 {
		response.Error(w, http.StatusBadRequest, "query too long")
		return
	}

	items, err := h.collegeService.Search(r.Context(), q)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	if err := h.stampSearchRepresented(r.Context(), items); err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	if uid, ok := auth.OptionalUserID(r.Context()); ok && len(items) > 0 {
		ids := make([]string, len(items))
		for i, it := range items {
			ids[i] = it.ID
		}
		if set, err := h.favorites.FavoritedCollegeIDs(r.Context(), uid, ids); err == nil {
			for i := range items {
				if _, ok := set[items[i].ID]; ok {
					items[i].IsFavorited = true
				}
			}
		}
	}

	response.Success(w, http.StatusOK, pagination.ItemsResponse[CollegeSearchResult]{Items: items})
}

func isAdmin(ctx context.Context) bool {
	claims, err := auth.ClaimsFromContext(ctx)
	if err != nil {
		return false
	}
	return claims.Role == auth.RoleAdmin
}
