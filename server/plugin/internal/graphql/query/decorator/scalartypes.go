package decorator

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/shurcooL/githubv4"
)

type typeDecorator func(string) func(*reflect.Type)

// DecorateScalarType loops through the given decorator functions and sets the correct value for reflect.Type of
// the Scalar item according to the Scalar.kind value of the item.
//
// Decorators must return a typeDecorator in order to work.
// If a new decorator is implemented, it must be added to the decorators list.
// If a decorator is no longer used, it must be taken out of the decorators list before being deleted.
func DecorateScalarType(kind string, rt *reflect.Type) error {
	decorators := []typeDecorator{
		stringType,
		intType,
		idType,
		booleanType,
		floatType,
		uriType,
		htmlType,
		dateType,
		datetimeType,
		mergeableStateType,
		pullRequestStateType,
		pullRequestReviewDecisionType,
	}
	for _, d := range decorators {
		d(strings.ToLower(kind))(rt)
	}
	if *rt == reflect.Type(nil) {
		return fmt.Errorf("scalar type not found: %s", kind)
	}

	return nil
}

func stringType(key string) func(*reflect.Type) {
	return func(t *reflect.Type) {
		if key == "string" {
			*t = reflect.TypeOf((*githubv4.String)(nil)).Elem()
		}
	}
}

func intType(key string) func(*reflect.Type) {
	return func(t *reflect.Type) {
		if key == "int" {
			*t = reflect.TypeOf((*githubv4.Int)(nil)).Elem()
		}
	}
}

func idType(key string) func(*reflect.Type) {
	return func(t *reflect.Type) {
		if key == "id" {
			*t = reflect.TypeOf((*githubv4.ID)(nil)).Elem()
		}
	}
}

func booleanType(key string) func(*reflect.Type) {
	return func(t *reflect.Type) {
		if key == "boolean" {
			*t = reflect.TypeOf((*githubv4.Boolean)(nil)).Elem()
		}
	}
}

func floatType(key string) func(*reflect.Type) {
	return func(t *reflect.Type) {
		if key == "float" {
			*t = reflect.TypeOf((*githubv4.Float)(nil)).Elem()
		}
	}
}

func uriType(key string) func(*reflect.Type) {
	return func(t *reflect.Type) {
		if key == "uri" {
			*t = reflect.TypeOf((*githubv4.URI)(nil)).Elem()
		}
	}
}

func htmlType(key string) func(*reflect.Type) {
	return func(t *reflect.Type) {
		if key == "html" {
			*t = reflect.TypeOf((*githubv4.HTML)(nil)).Elem()
		}
	}
}

func dateType(key string) func(*reflect.Type) {
	return func(t *reflect.Type) {
		if key == "date" {
			*t = reflect.TypeOf((*githubv4.Date)(nil)).Elem()
		}
	}
}

func datetimeType(key string) func(*reflect.Type) {
	return func(t *reflect.Type) {
		if key == "datetime" {
			*t = reflect.TypeOf((*githubv4.DateTime)(nil)).Elem()
		}
	}
}

func mergeableStateType(key string) func(*reflect.Type) {
	return func(t *reflect.Type) {
		if key == "mergeablestate" {
			*t = reflect.TypeOf((*githubv4.MergeableState)(nil)).Elem()
		}
	}
}

func pullRequestStateType(key string) func(*reflect.Type) {
	return func(t *reflect.Type) {
		if key == "pullrequeststate" {
			*t = reflect.TypeOf((*githubv4.PullRequestState)(nil)).Elem()
		}
	}
}

func pullRequestReviewDecisionType(key string) func(*reflect.Type) {
	return func(t *reflect.Type) {
		if key == "pullrequestreviewdecision" {
			*t = reflect.TypeOf((*githubv4.PullRequestReviewDecision)(nil)).Elem()
		}
	}
}
