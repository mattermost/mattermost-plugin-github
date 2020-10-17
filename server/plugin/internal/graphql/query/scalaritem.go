package query

import (
	"fmt"
	"strings"
)

type ScalarItem struct {
	name string
	kind string
}

func NewScalarItem(name, kind string) (ScalarItem, error) {
	if err := validKey(name); err != nil {
		return ScalarItem{}, fmt.Errorf("error setting 'key': %s", err.Error())
	}
	if err := validKey(kind); err != nil {
		return ScalarItem{}, fmt.Errorf("error setting 'kind': %s", err.Error())
	}

	name = strings.TrimSpace(strings.Title(name))
	if strings.ToLower(name) == "id" {
		name = "ID"
	}

	kind = strings.TrimSpace(strings.Title(kind))
	if strings.ToLower(kind) == "id" {
		kind = "ID"
	}

	return ScalarItem{name: name, kind: kind}, nil
}
