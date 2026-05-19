package rates

import (
	"fmt"
	"regexp"
	"strings"
)

var pairRe = regexp.MustCompile(`^[A-Z]{3}_[A-Z]{3}$`)

func NormalizePair(pair string) string {
	return strings.ToUpper(strings.TrimSpace(pair))
}

func ValidatePair(pair string) error {
	pair = NormalizePair(pair)
	if !pairRe.MatchString(pair) {
		return fmt.Errorf("invalid pair format: %q (expected AAA_BBB)", pair)
	}
	return nil
}

