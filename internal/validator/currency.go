package validator

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	validatorv10 "github.com/go-playground/validator/v10"
)

type supportedCountriesPayload struct {
	Countries []string `json:"countries"`
}

func New(countriesFilePath string) (*validatorv10.Validate, error) {
	countries, err := loadSupportedCountries(countriesFilePath)
	if err != nil {
		return nil, err
	}

	validate := validatorv10.New()

	iso4217Re := regexp.MustCompile(`^[A-Z]{3}$`)

	if err := validate.RegisterValidation("iso4217", func(fieldLevel validatorv10.FieldLevel) bool {
		currency := strings.ToUpper(strings.TrimSpace(fieldLevel.Field().String()))
		// For dynamic provider-driven rates we accept any ISO-like 3-letter code.
		// Pair-level support is enforced by the rates cache (rate:{pair}) existence.
		return iso4217Re.MatchString(currency)
	}); err != nil {
		return nil, fmt.Errorf("failed to register iso4217 validator: %w", err)
	}

	if err := validate.RegisterValidation("supported_country", func(fieldLevel validatorv10.FieldLevel) bool {
		country := strings.TrimSpace(fieldLevel.Field().String())
		_, exists := countries[country]
		return exists
	}); err != nil {
		return nil, fmt.Errorf("failed to register supported_country validator: %w", err)
	}

	return validate, nil
}

func loadSupportedCountries(countriesFilePath string) (map[string]struct{}, error) {
	content, err := os.ReadFile(countriesFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read supported countries file %q: %w", countriesFilePath, err)
	}

	var payload supportedCountriesPayload
	if err := json.Unmarshal(content, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse supported countries file %q: %w", countriesFilePath, err)
	}

	countries := make(map[string]struct{}, len(payload.Countries))
	for _, country := range payload.Countries {
		trimmed := strings.TrimSpace(country)
		if trimmed == "" {
			continue
		}
		countries[trimmed] = struct{}{}
	}

	return countries, nil
}
