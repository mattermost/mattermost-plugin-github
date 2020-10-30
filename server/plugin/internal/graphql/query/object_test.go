package query

import (
	"reflect"
	"testing"

	"github.com/shurcooL/githubv4"
)

func TestNewObject_SetFirst(t *testing.T) {
	type args struct {
		name string
		opts []Option
	}

	tests := []struct {
		name    string
		args    args
		want    *Object
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				name: "Test",
				opts: []Option{SetFirst(10)},
			},

			want: &Object{
				name: "Test",
				tag: tag{
					"first": 10,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid/value",
			args: args{
				name: "Test",
				opts: []Option{SetFirst(-1)},
			},

			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid/contains last",
			args: args{
				name: "Test",
				opts: []Option{
					SetLast(10),
					SetFirst(10),
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewObject(tt.args.name, tt.args.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewObject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewObject() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestNewObject_SetLast(t *testing.T) {
	type args struct {
		name string
		opts []Option
	}

	tests := []struct {
		name    string
		args    args
		want    *Object
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				name: "Test",
				opts: []Option{SetLast(10)},
			},
			want: &Object{
				name: "Test",
				tag: tag{
					"last": 10,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid/value",
			args: args{
				name: "Test",
				opts: []Option{SetLast(-1)},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid/contains first",
			args: args{
				name: "Test",
				opts: []Option{

					SetFirst(10),
					SetLast(10),
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewObject(tt.args.name, tt.args.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewObject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewObject() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestNewObject_SetBefore(t *testing.T) {
	type args struct {
		name string
		opt  Option
	}

	tests := []struct {
		name    string
		args    args
		want    *Object
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				name: "Test",
				opt:  SetBefore("test"),
			},
			want: &Object{
				name: "Test",
				tag:  tag{"before": "test"},
			},
			wantErr: false,
		},
		{
			name: "invalid",
			args: args{
				name: "Test",
				opt:  SetBefore("  "),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewObject(tt.args.name, tt.args.opt)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewObject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewObject() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestNewObject_SetAfter(t *testing.T) {
	type args struct {
		name string
		opt  Option
	}

	tests := []struct {
		name    string
		args    args
		want    *Object
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				name: "Search",
				opt:  SetAfter("test"),
			},
			want: &Object{
				name: "Search",
				tag:  tag{"after": "test"},
			},
			wantErr: false,
		},
		{
			name: "invalid",
			args: args{
				name: "Search",
				opt:  SetAfter("  "),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewObject(tt.args.name, tt.args.opt)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewObject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewObject() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestNewObject_SetSearchType(t *testing.T) {
	type args struct {
		name string
		opt  Option
	}
	tests := []struct {
		name    string
		args    args
		want    *Object
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				name: "Test",
				opt:  SetSearchType("issue"),
			},
			want: &Object{
				name: "Test",
				tag: tag{
					"type": githubv4.SearchTypeIssue,
				},
			},
			wantErr: false,
		},
		{
			name: "valid/white space",
			args: args{
				name: "Test",
				opt:  SetSearchType(" issue"),
			},
			want: &Object{
				name: "Test",
				tag: tag{
					"type": githubv4.SearchTypeIssue,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid",
			args: args{
				name: "Test",
				opt:  SetSearchType("test"),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewObject(tt.args.name, tt.args.opt)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewObject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewObject() got = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestNewObject(t *testing.T) {
	type args struct {
		name string
		opts []Option
	}
	tests := []struct {
		name    string
		args    args
		want    *Object
		wantErr bool
	}{
		{
			name:    "valid/name starts with lower case",
			args:    args{name: "pullRequest"},
			want:    &Object{name: "PullRequest", tag: make(tag, 1)},
			wantErr: false,
		},
		{
			name:    "invalid/empty name",
			args:    args{name: "  "},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid/name contains symbols",
			args:    args{name: "pull-request_test"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid/name contains digits",
			args:    args{name: "pullrequest10"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid/name contains space",
			args:    args{name: "pull request"},
			want:    nil,
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				name: "PullRequest",
				opts: []Option{
					SetFirst(10),
					SetSearchType("issue"),
					SetQuery("author:test is:pr is:OPEN archived:false"),
				},
			},
			want: &Object{
				name: "PullRequest",
				tag: tag{
					"first": 10,
					"type":  githubv4.SearchTypeIssue,
					"query": "author:test is:pr is:OPEN archived:false",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewObject(tt.args.name, tt.args.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewObject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewObject() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestObject_tagExists(t *testing.T) {
	tests := []struct {
		name string
		tag  tag
		key  string
		want bool
	}{
		{
			name: "key found",
			tag: tag{
				"first": 1,
				"after": 2,
				"name":  3,
			},
			key:  "first",
			want: true,
		},
		{
			name: "key not found",
			tag: tag{
				"last":  1,
				"after": 2,
				"name":  3,
			},
			key:  "first",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Object{
				name: "test",
				tag:  tt.tag,
			}
			if got := c.tagExists(tt.key); got != tt.want {
				t.Errorf("tagExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestObject_AddScalar(t *testing.T) {
	s, _ := NewScalar("Body", "String")
	s2, _ := NewScalar("Title", "String")

	want := &Object{
		name:    "Search",
		scalars: []Scalar{s, s2},
		tag:     make(tag, 1),
	}

	got, _ := NewObject("Search")
	got.AddScalar(s)
	got.AddScalar(s2)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("AddScalar() = %+v\nwant: %+v", got, want)
	}
}

func TestObject_AddObject(t *testing.T) {
	rr := &Object{
		name: "ReviewRequests",
		scalars: []Scalar{
			{
				name: "TotalCount",
				kind: "Int",
			},
		},
		tag: tag{
			"first": 10,
		},
	}
	repo := &Object{
		name: "Repository",
		scalars: []Scalar{
			{
				name: "Name",
				kind: "String",
			},
		},
		tag: make(tag, 1),
	}

	want := &Object{
		name:    "Search",
		objects: []Object{*rr, *repo},
		tag: tag{
			"first": 100,
			"type":  githubv4.SearchTypeIssue,
		},
	}

	got, _ := NewObject("Search", SetFirst(100), SetSearchType("issue"))
	got.AddObject(rr)
	got.AddObject(repo)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("AddObject() = %+v\nwant: %+v", got, want)
	}
}

func TestObject_SetNode(t *testing.T) {
	// Check wrong type (node/node list) error
	o, _ := NewObject("Search")
	if err := o.SetNode(NewNodeList()); err == nil {
		t.Errorf("SetNode() expected error, got nil")
		return
	}

	s, _ := NewScalar("Body", "String")
	s2, _ := NewScalar("Title", "String")

	n := NewNode()
	n.AddScalar(s)
	n.AddScalar(s2)

	want := &Object{
		name: "Search",
		node: n,
		tag:  make(tag, 1),
	}

	got, _ := NewObject("Search")
	if err := got.SetNode(n); err != nil {
		t.Errorf("SetNode() err = %v", err)
		return
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("SetNode() = %+v\nwant: %+v", got, want)
	}
}

func TestObject_SetNodeList(t *testing.T) {
	// Check wrong type (node/node list) error
	o, _ := NewObject("Search")
	if err := o.SetNodeList(NewNode()); err == nil {
		t.Errorf("SetNodeList() expected error, got nil")
		return
	}

	s, _ := NewScalar("Body", "String")
	s2, _ := NewScalar("Title", "String")

	n := NewNodeList()
	n.AddScalar(s)
	n.AddScalar(s2)

	want := &Object{
		name:     "Search",
		nodeList: n,
		tag:      make(tag, 1),
	}

	got, _ := NewObject("Search")
	if err := got.SetNodeList(n); err != nil {
		t.Errorf("SetNodeList() err = %v", err)
		return
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("SetNodeList() = %+v\nwant: %+v", got, want)
	}
}

func TestObject_AddUnion(t *testing.T) {
	got, _ := NewObject("Search")
	u, _ := NewUnion("PullRequest")
	got.AddUnion(u)

	want := &Object{
		name:   "Search",
		unions: []Union{*u},
		tag:    make(tag, 1),
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("AddUnion() = %T\nwant: %T", got, want)
	}
}

func TestNewObject_SetOrg(t *testing.T) {
	got, _ := NewObject("Test", SetOrg("github"))
	want := &Object{
		name: "Test",
		tag: tag{
			"org": "github",
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("SetOrg() = %v, want %v", got, want)
	}
}
