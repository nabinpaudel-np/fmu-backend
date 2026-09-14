package university

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"fmu-backend/internal/auth"
	"fmu-backend/internal/bulkimport"
	"fmu-backend/internal/errs"
	"fmu-backend/internal/pagination"
	"fmu-backend/internal/programs"
	"fmu-backend/internal/response"
	"fmu-backend/internal/validator"
)

// favoritesLookup is the minimal favorites dependency the public list/search
// handlers need to stamp `is_favorited`. Defined inline so this package only
// imports what it uses — favorites.Repository satisfies it implicitly.
type favoritesLookup interface {
	FavoritedUniversityIDs(ctx context.Context, userID string, ids []string) (map[string]struct{}, error)
}

// programsLookup is the minimal programs dependency the /lookups handler
// needs to bundle the programs list. programs.ProgramService satisfies it.
type programsLookup interface {
	ListAll(ctx context.Context) ([]programs.ProgramLookupItem, error)
}

type UniversityHandler struct {
	universityService UniversityService
	favorites         favoritesLookup
	programs          programsLookup
}

func NewUniversityHandler(universityService UniversityService, favs favoritesLookup, progs programsLookup) *UniversityHandler {
	return &UniversityHandler{
		universityService: universityService,
		favorites:         favs,
		programs:          progs,
	}
}

// stampFavorited sets IsFavorited on each item in-place. No-op for anonymous
// requests (OptionalAuthMiddleware didn't inject claims) and on lookup errors
// — the field defaults to false in both cases.
func (h *UniversityHandler) stampFavorited(ctx context.Context, items []UniversityListItem) {
	uid, ok := auth.OptionalUserID(ctx)
	if !ok || len(items) == 0 {
		return
	}
	ids := make([]string, len(items))
	for i, it := range items {
		ids[i] = it.ID
	}
	set, err := h.favorites.FavoritedUniversityIDs(ctx, uid, ids)
	if err != nil {
		return
	}
	for i := range items {
		if _, ok := set[items[i].ID]; ok {
			items[i].IsFavorited = true
		}
	}
}

func (h *UniversityHandler) stampRepresented(ctx context.Context, items []UniversityListItem) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]string, len(items))
	for i, item := range items {
		ids[i] = item.ID
	}
	set, err := h.universityService.RepresentedIDs(ctx, ids)
	if err != nil {
		return err
	}
	for i := range items {
		_, items[i].HasRepresentative = set[items[i].ID]
	}
	return nil
}

func (h *UniversityHandler) stampSearchRepresented(ctx context.Context, items []UniversitySearchResult) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]string, len(items))
	for i, item := range items {
		ids[i] = item.ID
	}
	set, err := h.universityService.RepresentedIDs(ctx, ids)
	if err != nil {
		return err
	}
	for i := range items {
		_, items[i].HasRepresentative = set[items[i].ID]
	}
	return nil
}

// resourceField maps a lookup-table name to the request-DTO field that
// references it, so error messages name the field the client sent.
var resourceField = map[string]string{
	"degree_levels":        "degree_level_ids",
	"majors":               "major_ids",
	"study_formats":        "study_format_ids",
	"special_affiliations": "special_affiliation_ids",
	"athletics":            "athletic_ids",
	"support_services":     "support_service_ids",
}

// formatMissingIDs caps the list at 10 IDs so a payload with hundreds of
// bad IDs doesn't bloat the error response.
func formatMissingIDs(ids []string) string {
	const cap = 10
	if len(ids) <= cap {
		return strings.Join(ids, ", ")
	}
	return strings.Join(ids[:cap], ", ") + fmt.Sprintf(" (and %d more)", len(ids)-cap)
}

func (h *UniversityHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateUniversityRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Drafts skip the full required-field validation — only name + slug
	// are required for drafts (enforced in the service). The DTO carries
	// `validate:"required"` on the rest, so we have to bypass the strict
	// pass when the caller is explicitly asking for a draft.
	if req.Status != "draft" {
		if err := validator.Validate.Struct(req); err != nil {
			validationErrors := validator.GetValidationErrors(err)
			response.ValidationError(w, http.StatusBadRequest, validationErrors)
			return
		}
	}

	res, err := h.universityService.Create(r.Context(), &req)
	if err != nil {
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
				details = append(details, response.ErrorDetail{
					Field:   f,
					Message: "required",
				})
			}
			response.ValidationError(w, http.StatusBadRequest, details)
			return
		}
		if errors.Is(err, errs.ErrUniversitySlugTaken) {
			response.Error(w, http.StatusConflict, err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	response.Success(w, http.StatusCreated, res)
}

// BulkUpload handles POST /api/v1/universities/bulk. Accepts a multipart
// upload with `file` (the CSV) and an optional `status` form field
// ("draft" or "published", defaults to "draft"). The whole upload is
// all-or-nothing: if any row fails validation the response is a 400 with
// the per-row error list and nothing is inserted. Rows whose slug is
// already in the DB are silently skipped (idempotency for the
// "append-and-reupload" workflow).
func (h *UniversityHandler) BulkUpload(w http.ResponseWriter, r *http.Request) {
	status, err := bulkimport.ValidateStatus(r.FormValue("status"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, bulkimport.MaxCSVBytes)
	if err := r.ParseMultipartForm(bulkimport.MaxCSVBytes); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			response.Error(w, http.StatusRequestEntityTooLarge,
				fmt.Sprintf("file exceeds %d bytes", bulkimport.MaxCSVBytes))
			return
		}
		response.Error(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "file is required (multipart field 'file')")
		return
	}
	defer file.Close()

	if header.Size <= 0 {
		response.Error(w, http.StatusBadRequest, "file is empty")
		return
	}

	head := make([]byte, 512)
	n, _ := io.ReadFull(file, head)
	mime := http.DetectContentType(head[:n])
	if !strings.HasPrefix(mime, "text/") && mime != "application/csv" && mime != "application/octet-stream" {
		response.Error(w, http.StatusUnsupportedMediaType,
			fmt.Sprintf("mime type %s is not allowed (expected CSV)", mime))
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	parsed := bulkimport.Parse(file, status)
	if len(parsed.Errors) > 0 {
		writeBulkValidationError(w, parsed.Errors)
		return
	}

	res, err := h.universityService.BulkUpload(r.Context(), status, parsed.Rows)
	if err != nil {
		var bulkErr *errs.BulkValidationError
		if errors.As(err, &bulkErr) {
			writeBulkValidationError(w, bulkErr.Errors)
			return
		}
		log.Default().Printf("bulk upload failed: %v", err)
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	response.Success(w, http.StatusOK, res)
}

// writeBulkValidationError emits the bulk-upload-specific 400 envelope.
// The shape is {created, skipped_existing, errors[]} rather than the
// generic APIResponse {success, errors[]} so the admin client can render
// the row/column/value triple without re-shaping it.
func writeBulkValidationError(w http.ResponseWriter, rowErrs []errs.RowError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(&BulkUploadResponse{
		Created:         0,
		SkippedExisting: 0,
		Errors:          toUniversityRowErrors(rowErrs),
	})
}

func toUniversityRowErrors(in []errs.RowError) []RowError {
	out := make([]RowError, len(in))
	for i, e := range in {
		out[i] = RowError{Row: e.Row, Column: e.Column, Value: e.Value, Message: e.Message}
	}
	return out
}

func (h *UniversityHandler) Publish(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	res, err := h.universityService.Publish(r.Context(), id)
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
			response.Error(w, http.StatusNotFound, "university not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	response.Success(w, http.StatusOK, res)
}

func (h *UniversityHandler) Patch(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req PatchUniversityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validator.Validate.Struct(&req); err != nil {
		validationErrors := validator.GetValidationErrors(err)
		response.ValidationError(w, http.StatusBadRequest, validationErrors)
		return
	}

	res, err := h.universityService.Patch(r.Context(), id, &req)
	if err != nil {
		if errors.Is(err, errs.ErrRepCannotChangeNameOrSlug) {
			response.Error(w, http.StatusForbidden, err.Error())
			return
		}
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
		if errors.Is(err, errs.ErrUniversitySlugTaken) {
			response.Error(w, http.StatusConflict, err.Error())
			return
		}
		if errors.Is(err, errs.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "university not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	response.Success(w, http.StatusOK, res)
}

func (h *UniversityHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	detail, err := h.universityService.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "university not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	// Non-admin callers see only published universities. Drafts and
	// archived rows are 404'd so they don't leak existence.
	if !isAdmin(r.Context()) && detail.Status != "published" {
		response.Error(w, http.StatusNotFound, "university not found")
		return
	}

	set, err := h.universityService.RepresentedIDs(r.Context(), []string{detail.ID})
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	_, detail.HasRepresentative = set[detail.ID]

	response.Success(w, http.StatusOK, detail)
}

func (h *UniversityHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		response.Error(w, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}
	if len(q) > 200 {
		response.Error(w, http.StatusBadRequest, "query too long")
		return
	}

	items, err := h.universityService.Search(r.Context(), q)
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
		if set, err := h.favorites.FavoritedUniversityIDs(r.Context(), uid, ids); err == nil {
			for i := range items {
				if _, ok := set[items[i].ID]; ok {
					items[i].IsFavorited = true
				}
			}
		}
	}

	response.Success(w, http.StatusOK, pagination.ItemsResponse[UniversitySearchResult]{Items: items})
}

func (h *UniversityHandler) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.universityService.Stats(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	response.Success(w, http.StatusOK, stats)
}

func (h *UniversityHandler) Get(w http.ResponseWriter, r *http.Request) {
	q := pagination.Parse(r)
	filters := ParseFilters(r.URL.Query())

	// Non-admin callers can only see published rows. We silently narrow the
	// status filter to "published" so an attempted `?status=draft` from a
	// public client returns the public list rather than a 403.
	if !isAdmin(r.Context()) {
		filters.Status = "published"
	}

	items, total, err := h.universityService.Get(r.Context(), q, filters)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	if err := h.stampRepresented(r.Context(), items); err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	h.stampFavorited(r.Context(), items)

	response.Success(w, http.StatusOK, pagination.Response[UniversityListItem]{
		Items: items,
		Meta:  q.BuildMeta(total),
	})
}

func isAdmin(ctx context.Context) bool {
	claims, err := auth.ClaimsFromContext(ctx)
	if err != nil {
		return false
	}
	return claims.Role == auth.RoleAdmin
}

func (h *UniversityHandler) GetMajors(w http.ResponseWriter, r *http.Request) {
	majors, err := h.universityService.GetMajors(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	response.Success(w, http.StatusOK, pagination.ItemsResponse[MajorResponse]{Items: majors})
}

func (h *UniversityHandler) GetDegreeLevels(w http.ResponseWriter, r *http.Request) {
	items, err := h.universityService.GetDegreeLevels(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	response.Success(w, http.StatusOK, pagination.ItemsResponse[DegreeLevelResponse]{Items: items})
}

func (h *UniversityHandler) GetStudyFormats(w http.ResponseWriter, r *http.Request) {
	items, err := h.universityService.GetStudyFormats(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	response.Success(w, http.StatusOK, pagination.ItemsResponse[StudyFormatResponse]{Items: items})
}

func (h *UniversityHandler) GetSpecialAffiliations(w http.ResponseWriter, r *http.Request) {
	items, err := h.universityService.GetSpecialAffiliations(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	response.Success(w, http.StatusOK, pagination.ItemsResponse[SpecialAffiliationResponse]{Items: items})
}

func (h *UniversityHandler) GetAthletics(w http.ResponseWriter, r *http.Request) {
	items, err := h.universityService.GetAthletics(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	response.Success(w, http.StatusOK, pagination.ItemsResponse[AthleticResponse]{Items: items})
}

func (h *UniversityHandler) GetSupportServices(w http.ResponseWriter, r *http.Request) {
	items, err := h.universityService.GetSupportServices(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	response.Success(w, http.StatusOK, pagination.ItemsResponse[SupportServiceResponse]{Items: items})
}

func (h *UniversityHandler) GetAllLookups(w http.ResponseWriter, r *http.Request) {
	lookups, err := h.universityService.GetAllLookups(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	// Bundle programs alongside the other reference lists. Best-effort: if
	// the lookup fails the rest still serves, just without programs.
	if progs, err := h.programs.ListAll(r.Context()); err == nil {
		lookups.Programs = progs
	}
	response.Success(w, http.StatusOK, lookups)
}
