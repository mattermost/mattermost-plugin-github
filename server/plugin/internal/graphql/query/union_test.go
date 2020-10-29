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
		tag:  tag{},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("NewUnion() got = %+v, want %+v", got, want)
	}
}

func TestUnion_AddObject(t *testing.T) {
	o, _ := NewObject("Search", SetFirst(100))
	want := &Union{
		name:    "PullRequest",
		objects: []Object{*o},
		tag:     tag{},
	}

	got, _ := NewUnion("PullRequest")
	got.AddObject(o)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("AddObject() = %+v\nwant: %+v", got, want)
	}
}

func TestUnion_AddScalar(t *testing.T) {
	s, _ := NewScalar("Body", "String")
	s2, _ := NewScalar("Title", "String")

	want := &Union{
		name:    "PullRequest",
		scalars: []Scalar{s, s2},
		tag:     tag{},
	}

	got, _ := NewUnion("PullRequest")
	got.AddScalar(s)
	got.AddScalar(s2)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("AddScalar() = %+v\nwant: %+v", got, want)
	}
}
