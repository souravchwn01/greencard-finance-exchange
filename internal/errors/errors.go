package errors

import "net/http"

type DomainError struct {
	Code       string
	Message    string
	HTTPStatus int
}

func (e *DomainError) Error() string { return e.Message }

var (
	ErrMissingParameter = &DomainError{
		Code: "MISSING_PARAMETER", HTTPStatus: http.StatusBadRequest,
		Message: "A required query parameter is missing.",
	}
	ErrInvalidCurrency = &DomainError{
		Code: "INVALID_CURRENCY", HTTPStatus: http.StatusUnprocessableEntity,
		Message: "One or more currency codes are not supported.",
	}
	ErrSameCurrency = &DomainError{
		Code: "SAME_CURRENCY", HTTPStatus: http.StatusUnprocessableEntity,
		Message: "From and To currencies must be different.",
	}
	ErrInvalidCountry = &DomainError{
		Code: "INVALID_COUNTRY", HTTPStatus: http.StatusUnprocessableEntity,
		Message: "Sender Country is not in the supported list.",
	}
	ErrUnsupportedPair = &DomainError{
		Code: "UNSUPPORTED_CURRENCY_PAIR", HTTPStatus: http.StatusNotFound,
		Message: "No exchange rate is configured for this currency pair.",
	}
	ErrInternal = &DomainError{
		Code: "INTERNAL_ERROR", HTTPStatus: http.StatusInternalServerError,
		Message: "An unexpected error occurred.",
	}
)
