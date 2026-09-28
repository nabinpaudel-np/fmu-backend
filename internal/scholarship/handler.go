package scholarship

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

// favoritesLookup is the minimal favorites dependency the public list/search/
// detail handlers need to stamp `is_favorited`. Defined inline so this package
// only imports what it uses — favorites.Repository satisfies it implicitly.
type favoritesLookup interface {
	FavoritedScholarshipIDs(ctx context.Context, userID string, ids []string) (map[string]struct{}, error)
}

type ScholarshipHandler struct {
	svc       ScholarshipService
	favorites favoritesLookup
}

func NewScholarshipHandler(svc ScholarshipService, favs favoritesLookup) *ScholarshipHandler {
	return &ScholarshipHandler{svc: svc, favorites: favs}
}

// resourceField maps a lookup-table name to the request-DTO field that
// references it, so error messages name the field the client sent.
var resourceField = map[string]string{
	"education_levels": "education_level_ids",
	"majors":           "major_ids",
	"demographics":     "demographic_ids",
}

// formatMissingIDs caps the list at 10 IDs so a payload with hundreds of bad
// IDs doesn't bloat the error response.
func formatMissingIDs(ids []string) string {
	const cap = 10
	if len(ids) <= cap {
		return strings.Join(ids, ", ")
	}
	return strings.Join(ids[:cap], ", ") + fmt.Sprintf(" (and %d more)", len(ids)-cap)
}

func isAdmin(ctx context.Context) bool {
	claims, err := auth.ClaimsFromContext(ctx)
	if err != nil {
		return false
	}
	return claims.Role == auth.RoleAdmin
}

// writeMutationError maps service errors shared by Create/Update to HTTP.
func writeMutationError(w http.ResponseWriter, err error) {
	var refErr *errs.InvalidReferencesError
	if errors.As(err, &refErr) {
		details := make([]response.ErrorDetail, 0, len(refErr.References))
		for resource, ids := range refErr.References {
			details = append(details, response.ErrorDetail{
				Field:   resourceField[resource],
				Message: fmt.Sprintf("the following %s do not exist: [%s]", resource, formatMissingIDs(ids)),
			})
		}
		response.ValidationError(w, http.StatusBadRequest, details)
		return
	}
	var pubErr *errs.PublishValidationError
	if errors.As(err, &pubErr) {
		details := make([]response.ErrorDetail, 0, len(pubErr.Fields))
		for _, f := range pubErr.Fields {
			details = append(details, response.ErrorDetail{Field: f, Message: "required"})
		}
		response.ValidationError(w, http.StatusBadRequest, details)
		return
	}
	if errors.Is(err, errs.ErrScholarshipSlugTaken) {
		response.Error(w, http.StatusConflict, err.Error())
		return
	}
	if errors.Is(err, errs.ErrScholarshipProviderNotFound) {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if errors.Is(err, errs.ErrNotFound) {
		response.Error(w, http.StatusNotFound, "scholarship not found")
		return
	}
	response.Error(w, http.StatusInternalServerError, "something went wrong")
}

func (h *ScholarshipHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateScholarshipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Drafts skip the full required-field validation — only title + slug are
	// required (enforced in the service). Publish re-validates before going live.
	if req.Status != "draft" {
		if err := validator.Validate.Struct(&req); err != nil {
			response.ValidationError(w, http.StatusBadRequest, validator.GetValidationErrors(err))
			return
		}
	}

	res, err := h.svc.Create(r.Context(), &req)
	if err != nil {
		writeMutationError(w, err)
		return
	}
	response.Success(w, http.StatusCreated, res)
}

func (h *ScholarshipHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req UpdateScholarshipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validator.Validate.Struct(&req); err != nil {
		response.ValidationError(w, http.StatusBadRequest, validator.GetValidationErrors(err))
		return
	}

	res, err := h.svc.Update(r.Context(), id, &req)
	if err != nil {
		writeMutationError(w, err)
		return
	}
	response.Success(w, http.StatusOK, res)
}

func (h *ScholarshipHandler) Publish(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	res, err := h.svc.Publish(r.Context(), id)
	if err != nil {
		var pubErr *errs.PublishValidationError
		if errors.As(err, &pubErr) {
			details := make([]response.ErrorDetail, 0, len(pubErr.Fields))
			for _, f := range pubErr.Fields {
				details = append(details, response.ErrorDetail{Field: f, Message: "required to publish"})
			}
			response.ValidationError(w, http.StatusBadRequest, details)
			return
		}
		if errors.Is(err, errs.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "scholarship not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	response.Success(w, http.StatusOK, res)
}

func (h *ScholarshipHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "scholarship not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ScholarshipHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	detail, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "scholarship not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	// Non-admin callers see only published scholarships; drafts/archived rows
	// are 404'd so they don't leak existence, and internal_notes is stripped.
	if !isAdmin(r.Context()) {
		if detail.Status != "published" {
			response.Error(w, http.StatusNotFound, "scholarship not found")
			return
		}
		detail.InternalNotes = nil
	}

	if uid, ok := auth.OptionalUserID(r.Context()); ok {
		if set, err := h.favorites.FavoritedScholarshipIDs(r.Context(), uid, []string{detail.ID}); err == nil {
			_, detail.IsFavorited = set[detail.ID]
		}
	}

	response.Success(w, http.StatusOK, detail)
}

func (h *ScholarshipHandler) Get(w http.ResponseWriter, r *http.Request) {
	q := pagination.Parse(r)
	filters := ParseFilters(r.URL.Query())

	// Non-admin callers can only see published rows. Silently narrow the status
	// filter so an attempted ?status=draft returns the public list, not a 403.
	if !isAdmin(r.Context()) {
		filters.Status = "published"
	}

	items, total, err := h.svc.Get(r.Context(), q, filters)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	h.stampFavorited(r.Context(), items)

	response.Success(w, http.StatusOK, pagination.Response[ScholarshipListItem]{
		Items: items,
		Meta:  q.BuildMeta(total),
	})
}

// stampFavorited sets IsFavorited on each item in-place. No-op for anonymous
// requests and on lookup errors — the field defaults to false.
func (h *ScholarshipHandler) stampFavorited(ctx context.Context, items []ScholarshipListItem) {
	uid, ok := auth.OptionalUserID(ctx)
	if !ok || len(items) == 0 {
		return
	}
	ids := make([]string, len(items))
	for i, it := range items {
		ids[i] = it.ID
	}
	set, err := h.favorites.FavoritedScholarshipIDs(ctx, uid, ids)
	if err != nil {
		return
	}
	for i := range items {
		if _, ok := set[items[i].ID]; ok {
			items[i].IsFavorited = true
		}
	}
}

func (h *ScholarshipHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		response.Error(w, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}
	if len(q) > 200 {
		response.Error(w, http.StatusBadRequest, "query too long")
		return
	}

	items, err := h.svc.Search(r.Context(), q)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	if uid, ok := auth.OptionalUserID(r.Context()); ok && len(items) > 0 {
		ids := make([]string, len(items))
		for i, it := range items {
			ids[i] = it.ID
		}
		if set, err := h.favorites.FavoritedScholarshipIDs(r.Context(), uid, ids); err == nil {
			for i := range items {
				if _, ok := set[items[i].ID]; ok {
					items[i].IsFavorited = true
				}
			}
		}
	}

	response.Success(w, http.StatusOK, pagination.ItemsResponse[ScholarshipSearchResult]{Items: items})
}

func (h *ScholarshipHandler) GetAllLookups(w http.ResponseWriter, r *http.Request) {
	lookups, err := h.svc.GetAllLookups(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	response.Success(w, http.StatusOK, lookups)
}

func (h *ScholarshipHandler) GetEducationLevels(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.GetEducationLevels(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	response.Success(w, http.StatusOK, pagination.ItemsResponse[EducationLevelResponse]{Items: items})
}

func (h *ScholarshipHandler) GetDemographics(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.GetDemographics(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	response.Success(w, http.StatusOK, pagination.ItemsResponse[DemographicResponse]{Items: items})
}
