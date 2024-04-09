package serializer

import (
	"encoding/json"

	"github.com/google/go-github/v48/github"
)

type SidebarContent struct {
	PRs         []*github.Issue         `json:"prs"`
	Reviews     []*github.Issue         `json:"reviews"`
	Assignments []*github.Issue         `json:"assignments"`
	Unreads     []*FilteredNotification `json:"unreads"`
}

func (s *SidebarContent) ToMap() (map[string]interface{}, error) {
	var m map[string]interface{}
	bytes, err := json.Marshal(&s)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(bytes, &m); err != nil {
		return nil, err
	}

	return m, nil
}
