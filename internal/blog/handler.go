package blog

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"fmu-backend/internal/errs"
	"fmu-backend/internal/response"
	"fmu-backend/internal/validator"
)

type BlogHandler struct {
	svc BlogService
}

func NewBlogHandler(svc BlogService) *BlogHandler {
	return &BlogHandler{svc: svc}
}

// Create handles POST /api/v1/blogs. Admin-only.
func (h *BlogHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateBlogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validator.Validate.Struct(&req); err != nil {
		fields := validator.GetValidationErrors(err)
		response.ValidationError(w, http.StatusBadRequest, fields)
		return
	}
	res, err := h.svc.Create(r.Context(), &req)
	if err != nil {
		writeServiceErr(w, err)
		return
	}
	response.Success(w, http.StatusCreated, res)
}

// GetByID handles GET /api/v1/blogs/id/{id}. Admin-only.
func (h *BlogHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	res, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		writeServiceErr(w, err)
		return
	}
	response.Success(w, http.StatusOK, res)
}

// Update handles PUT/PATCH /api/v1/blogs/{id}. Admin-only. Status is NOT
// touched here; use POST /{id}/publish for status transitions.
func (h *BlogHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req UpdateBlogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validator.Validate.Struct(&req); err != nil {
		fields := validator.GetValidationErrors(err)
		response.ValidationError(w, http.StatusBadRequest, fields)
		return
	}
	res, err := h.svc.Update(r.Context(), id, &req)
	if err != nil {
		writeServiceErr(w, err)
		return
	}
	response.Success(w, http.StatusOK, res)
}

// Delete handles DELETE /api/v1/blogs/{id}. Admin-only. Hard delete
// (matches existing resources — there is no soft delete in this codebase).
func (h *BlogHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeServiceErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Publish handles POST /api/v1/blogs/{id}/publish with body
// `{"publish": true|false}`. Admin-only.
func (h *BlogHandler) Publish(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req PublishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	res, err := h.svc.Publish(r.Context(), id, req.Publish)
	if err != nil {
		writeServiceErr(w, err)
		return
	}
	response.Success(w, http.StatusOK, res)
}

// ListAdmin handles GET /api/v1/blogs/all?status=&page=&page_size=.
// Admin-only. Returns all statuses (draft, published, archived) unless a
// status filter is supplied. Includes full body_html in each item.
func (h *BlogHandler) ListAdmin(w http.ResponseWriter, r *http.Request) {
	items, total, err := h.svc.ListAdmin(r.Context(), parseFilters(r))
	if err != nil {
		writeServiceErr(w, err)
		return
	}
	page, limit := parsePaging(r)
	response.Success(w, http.StatusOK, BlogListResponse{
		Items:      items,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: pages(total, limit),
	})
}

// ListPublic handles GET /api/v1/blogs. Public (no auth). Returns only
// published posts, omitting body_html to keep payloads small.
func (h *BlogHandler) ListPublic(w http.ResponseWriter, r *http.Request) {
	items, total, err := h.svc.ListPublic(r.Context(), parseFilters(r))
	if err != nil {
		writeServiceErr(w, err)
		return
	}
	page, limit := parsePaging(r)
	response.Success(w, http.StatusOK, BlogListResponse{
		Items:      items,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: pages(total, limit),
	})
}

// GetBySlug handles GET /api/v1/blogs/{slug}. Public (no auth). Returns
// the post only when status='published'; otherwise 404. The published-only
// filter is enforced in SQL via GetPublishedBlogBySlug.
func (h *BlogHandler) GetBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	res, err := h.svc.GetPublishedBySlug(r.Context(), slug)
	if err != nil {
		writeServiceErr(w, err)
		return
	}
	response.Success(w, http.StatusOK, res)
}

func parseFilters(r *http.Request) ListFilters {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit == 0 {
		limit, _ = strconv.Atoi(q.Get("page_size"))
	}
	return ListFilters{
		Status: q.Get("status"),
		// Accept either repeated query params (?tags=a&tags=b) or a single
		// comma-separated value (?tags=a,b). Both shapes are common across
		// REST clients; supporting them keeps callers from having to URL-encode
		// repeated keys just to filter by two tags.
		Tags:  parseTagParam(q["tags"]),
		Page:  page,
		Limit: limit,
	}
}

// parseTagParam flattens repeated query values into a single slice and
// also splits comma-separated values, trimming whitespace around each tag.
// Empty entries are dropped — they come from trailing/leading commas and
// add no signal to the filter.
func parseTagParam(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, v := range values {
		for _, piece := range strings.Split(v, ",") {
			if t := strings.TrimSpace(piece); t != "" {
				out = append(out, t)
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func parsePaging(r *http.Request) (int, int) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit == 0 {
		limit, _ = strconv.Atoi(q.Get("page_size"))
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}

func pages(total int64, limit int) int {
	if limit <= 0 || total <= 0 {
		return 0
	}
	return int((total + int64(limit) - 1) / int64(limit))
}

// writeServiceErr maps service-layer sentinel errors to HTTP status codes.
// All other errors are 500.
func writeServiceErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errs.ErrNotFound):
		response.Error(w, http.StatusNotFound, "blog not found")
	case errors.Is(err, errs.ErrBlogSlugTaken):
		response.Error(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrInvalidSlug):
		response.Error(w, http.StatusBadRequest, err.Error())
	default:
		response.Error(w, http.StatusInternalServerError, "something went wrong")
	}
}
