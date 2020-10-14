package query

import (
	"fmt"
	"regexp"
	"strings"
)

func greaterThan(val, lim int) error {
	if val <= lim {
		return fmt.Errorf("value cannot be less than %d", lim)
	}

	return nil
}

func strNotEmpty(val string) error {
	if strings.TrimSpace(val) == "" {
		return fmt.Errorf("value cannot be empty")
	}

	return nil
}

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
