package query

import (
	"reflect"
	"testing"

	"github.com/shurcooL/githubv4"
)

func TestNewObject_SetName(t *testing.T) {
	tests := []struct {
		name    string
		opt     Option
		want    *Object
		wantErr bool
	}{
		{
			name:    "valid",
			opt:     SetName("PullRequest"),
			want:    &Object{name: "PullRequest", tag: make(tag, 1)},
			wantErr: false,
		},
		{
			name:    "valid/initial starts with lower case",
			opt:     SetName("pullRequest"),
			want:    &Object{name: "PullRequest", tag: make(tag, 1)},
			wantErr: false,
		},
		{
			name:    "invalid/empty",
			opt:     SetName("  "),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid/contains symbols",
			opt:     SetName("pull-request_test"),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid/contains digits",
			opt:     SetName("pullrequest10"),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid/contains space",
			opt:     SetName("pull request"),
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewObject(tt.opt)
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

func TestNewObject_SetFirst(t *testing.T) {

	tests := []struct {
		name    string
		opts    []Option
		want    *Object
		wantErr bool
	}{
		{
			name: "valid",
			opts: []Option{SetFirst(10)},
			want: &Object{
				tag: tag{
					"first": 10,
				},
			},
			wantErr: false,
		},
		{
			name:    "invalid/value",
			opts:    []Option{SetFirst(-1)},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid/contains last",
			opts: []Option{
				SetLast(10),
				SetFirst(10),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewObject(tt.opts...)
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
	tests := []struct {
		name    string
		opts    []Option
		want    *Object
		wantErr bool
	}{
		{
			name: "valid",
			opts: []Option{SetLast(10)},
			want: &Object{
				tag: tag{
					"last": 10,
				},
			},
			wantErr: false,
		},
		{
			name:    "invalid/value",
			opts:    []Option{SetLast(-1)},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid/contains first",
			opts: []Option{
				SetFirst(10),
				SetLast(10),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewObject(tt.opts...)
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
	tests := []struct {
		name    string
		opt     Option
		want    *Object
		wantErr bool
	}{
		{
			name: "valid",
			opt:  SetBefore("test"),
			want: &Object{
				tag: tag{"before": "test"},
			},
			wantErr: false,
		},
		{
			name:    "invalid",
			opt:     SetBefore("  "),
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewObject(tt.opt)
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
	tests := []struct {
		name    string
		opt     Option
		want    *Object
		wantErr bool
	}{
		{
			name: "valid",
			opt:  SetAfter("test"),
			want: &Object{
				tag: tag{"after": "test"},
			},
			wantErr: false,
		},
		{
			name:    "invalid",
			opt:     SetAfter("  "),
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewObject(tt.opt)
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
	tests := []struct {
		name    string
		opt     Option
		want    *Object
		wantErr bool
	}{
		{
			name: "valid",
			opt:  SetSearchType("issue"),
			want: &Object{
				tag: tag{
					"type": githubv4.SearchTypeIssue,
				},
			},
			wantErr: false,
		},
		{
			name: "valid/white space",
			opt:  SetSearchType(" issue"),
			want: &Object{
				tag: tag{
					"type": githubv4.SearchTypeIssue,
				},
			},
			wantErr: false,
		},
		{
			name:    "invalid",
			opt:     SetSearchType("test"),
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewObject(tt.opt)
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
func TestNewObject_multipleOptions(t *testing.T) {
	type args struct {
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
				opts: []Option{
					SetName("PullRequest"),
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
					"query": "\"author:test is:pr is:OPEN archived:false\"",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewObject(tt.args.opts...)
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
