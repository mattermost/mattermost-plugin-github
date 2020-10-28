package query

import (
	"fmt"
	"regexp"
	"strings"
)

// greaterThan checks if the given value is greater than the limit, returns an error if not
func greaterThan(val, lim int) error {
	if val <= lim {
		return fmt.Errorf("value cannot be less than %d", lim)
	}

	return nil
}

// strNotEmpty checks if given string is empty, returns an error if it is
func strNotEmpty(val string) error {
	if strings.TrimSpace(val) == "" {
		return fmt.Errorf("value cannot be empty")
	}

	return nil
}

// validKey checks if val is not empty and has only non-alpha characters, returns an error if any of the checks fail.
func validKey(val string) error {
	if strNotEmpty(val) != nil {
		return fmt.Errorf("key cannot be empty")
	}

	re := regexp.MustCompile(`(?is)[\W_\d\s]`)
	if len(re.FindStringIndex(val)) > 0 {
		return fmt.Errorf("key cannot contain non-alpha characters")
	}

	return nil
}
