package query

import (
	"reflect"
	"testing"
)

func TestNewUnion(t *testing.T) {
	got, err := NewUnion("PullRequest")
	if err != nil {
		t.Errorf("NewUnion() error = %v", err)
		return
	}

	want := &Union{
		name: "PullRequest",
		tag: tag{
			TagKeyUnion: "... on PullRequest",
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("NewUnion() got = %+v, want %+v", got, want)
	}
}
