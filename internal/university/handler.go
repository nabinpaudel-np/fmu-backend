package university

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"fmu-backend/internal/errs"
	"fmu-backend/internal/pagination"
	"fmu-backend/internal/response"
	"fmu-backend/internal/validator"
)

type UniversityHandler struct {
	universityService UniversityService
}

func NewUniversityHandler(universityService UniversityService) *UniversityHandler {
	return &UniversityHandler{
		universityService: universityService,
	}
}

// resourceField maps a lookup-table name to its request-DTO field name.
var resourceField = map[string]string{
	"degree_levels":        "degree_level_ids",
	"majors":               "major_ids",
	"study_formats":        "study_format_ids",
	"special_affiliations": "special_affiliation_ids",
	"athletics":            "athletic_ids",
	"support_services":     "support_service_ids",
}

// formatMissingIDs returns a human-readable list, capping at 10 IDs with
// a "(and N more)" suffix so a payload with hundreds of bad IDs does not
// blow up the response body.
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

	if err := validator.Validate.Struct(req); err != nil {
		validationErrors := validator.GetValidationErrors(err)
		response.ValidationError(w, http.StatusBadRequest, validationErrors)
		return
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
		if errors.Is(err, errs.ErrUniversitySlugTaken) {
			response.Error(w, http.StatusConflict, err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	response.Success(w, http.StatusCreated, res)
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

	response.Success(w, http.StatusOK, detail)
}

func (h *UniversityHandler) Get(w http.ResponseWriter, r *http.Request) {
	q := pagination.Parse(r)
	filters := ParseFilters(r.URL.Query())

	items, total, err := h.universityService.Get(r.Context(), q, filters)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	response.Success(w, http.StatusOK, pagination.Response[UniversityListItem]{
		Items: items,
		Meta:  q.BuildMeta(total),
	})
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
	response.Success(w, http.StatusOK, lookups)
}
