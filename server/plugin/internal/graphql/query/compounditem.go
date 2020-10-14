package query

import (
	"fmt"
	"strings"

	"github.com/shurcooL/githubv4"
)

type (
	CompoundItem struct {
		name  string
		items []ScalarItem
		nodes []Node
		tag   tag
	}

	Node struct {
		items         []ScalarItem
		compoundItems []CompoundItem
	}

	Option func(item *CompoundItem) error

	tag map[string]interface{}
)

func NewCompoundItem(opts ...Option) (*CompoundItem, error) {
	c := &CompoundItem{
		tag: make(tag),
	}

	for _, opt := range opts {
		err := opt(c)
		if err != nil {
			return nil, err
		}
	}

	return c, nil
}

func SetName(val string) Option {
	return func(item *CompoundItem) error {
		if err := validKey(val); err != nil {
			return err
		}
		item.name = strings.Title(strings.TrimSpace(val))
		return nil
	}
}

func SetFirst(val int) Option {
	return func(item *CompoundItem) error {
		if err := greaterThan(val, 0); err != nil {
			return err
		}

		if item.tagExists("last") {
			return fmt.Errorf("cannot use 'first' and 'last' at the same time")
		}

		item.tag["first"] = val
		return nil
	}
}

func SetLast(val int) Option {
	return func(item *CompoundItem) error {
		if err := greaterThan(val, 0); err != nil {
			return err
		}

		if item.tagExists("first") {
			return fmt.Errorf("cannot use 'first' and 'last' at the same time")
		}

		item.tag["last"] = val
		return nil
	}
}

func SetBefore(val string) Option {
	return func(item *CompoundItem) error {
		if err := strNotEmpty(val); err != nil {
			return err
		}

		item.tag["before"] = val
		return nil
	}
}

func SetAfter(val string) Option {
	return func(item *CompoundItem) error {
		if err := strNotEmpty(val); err != nil {
			return err
		}

		item.tag["after"] = val
		return nil
	}
}

func SetQuery(val string) Option {
	return func(item *CompoundItem) error {
		if err := strNotEmpty(val); err != nil {
			return err
		}

		item.tag["query"] = "\"" + val + "\""
		return nil
	}
}

func SetSearchType(val string) Option {
	return func(item *CompoundItem) error {
		val = strings.ToUpper(strings.TrimSpace(val))
		var searchType githubv4.SearchType

		searchTypes := []githubv4.SearchType{
			githubv4.SearchTypeIssue,
			githubv4.SearchTypeUser,
			githubv4.SearchTypeRepository,
		}

		for _, v := range searchTypes {
			if val == string(v) {
				searchType = v
				break
			}
		}

		if searchType == "" {
			return fmt.Errorf("unexpected search type")
		}

		item.tag["type"] = searchType
		return nil
	}
}

func (c *CompoundItem) tagExists(key string) bool {
	for k, _ := range c.tag {
		if k == key {
			return true
		}
	}

	return false
}
