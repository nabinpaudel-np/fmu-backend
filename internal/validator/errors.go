package validator

import (
	"strings"

	"fmu-backend/internal/response"

	"github.com/go-playground/validator/v10"
)

func GetValidationErrors(err error) []response.ErrorDetail {
	if err == nil {
		return nil
	}

	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return []response.ErrorDetail{
			{Field: "error", Message: err.Error()},
		}
	}

	var errors []response.ErrorDetail

	for _, fe := range validationErrors {
		msgTemplate := errorMessages[fe.Tag()]
		if msgTemplate == "" {
			msgTemplate = "{0} is invalid"
		}

		message := strings.Replace(msgTemplate, "{0}", fe.Field(), 1)
		if strings.Contains(msgTemplate, "{1}") && fe.Param() != "" {
			message = strings.Replace(message, "{1}", fe.Param(), 1)
		}

		errors = append(errors, response.ErrorDetail{
			Field:   fe.Field(),
			Message: message,
		})
	}

	return errors
}

var errorMessages = map[string]string{
	"required": "{0} is required",
	"min":      "{0} must be at least {1} characters",
	"max":      "{0} must not exceed {1} characters",
	"email":    "{0} must be a valid email address",
	"len":      "{0} must be exactly {1} characters",
	"oneof":    "{0} must be one of: {1}",
	"gt":       "{0} must be greater than {1}",
	"gte":      "{0} must be at least {1}",
	"lt":       "{0} must be less than {1}",
	"lte":      "{0} must be at most {1}",
	"uuid":     "{0} must be a valid UUID",
}
