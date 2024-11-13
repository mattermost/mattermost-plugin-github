package plugin

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateFeatures(t *testing.T) {
	type output struct {
		valid           bool
		invalidFeatures []string
	}
	tests := []struct {
		name string
		args []string
		want output
	}{
		{
			name: "all features valid",
			args: []string{"creates", "pushes", "issue_comments"},
			want: output{true, []string{}},
		},
		{
			name: "all features invalid",
			args: []string{"create", "push"},
			want: output{false, []string{"create", "push"}},
		},
		{
			name: "first feature invalid",
			args: []string{"create", "pushes", "issue_comments"},
			want: output{false, []string{"create"}},
		},
		{
			name: "last feature invalid",
			args: []string{"creates", "push"},
			want: output{false, []string{"push"}},
		},
		{
			name: "multiple features invalid",
			args: []string{"create", "pushes", "issue"},
			want: output{false, []string{"create", "issue"}},
		},
		{
			name: "all features valid with label but issues and pulls missing",
			args: []string{"pushes", `label:"ruby"`},
			want: output{false, []string{}},
		},
		{
			name: "all features valid with label and issues in features",
			args: []string{"issues", `label:"ruby"`},
			want: output{true, []string{}},
		},
		{
			name: "all features valid with label and pulls in features",
			args: []string{"pulls", `label:"ruby"`},
			want: output{true, []string{}},
		},
		{
			name: "multiple features invalid with label but issues and pulls missing",
			args: []string{"issue", "push", `label:"ruby"`},
			want: output{false, []string{"issue", "push"}},
		},
		{
			name: "multiple features invalid with label and issues in features",
			args: []string{"issues", "push", "create", `label:"ruby"`},
			want: output{false, []string{"push", "create"}},
		},
		{
			name: "multiple features invalid with label and pulls in features",
			args: []string{"pulls", "push", "create", `label:"ruby"`},
			want: output{false, []string{"push", "create"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, fs := validateFeatures(tt.args)
			got := output{ok, fs}
			testFailureMessage := fmt.Sprintf("validateFeatures() = %v, want %v", got, tt.want)
			assert.EqualValues(t, tt.want, got, testFailureMessage)
		})
	}
}

func TestParseCommand(t *testing.T) {
	type output struct {
		command    string
		action     string
		parameters []string
	}

	tt := []struct {
		name  string
		input string
		want  output
	}{
		{
			name:  "no parameters",
			input: "/forgejo subscribe",
			want: output{
				"/forgejo",
				"subscribe",
				[]string(nil),
			},
		},
		{
			name:  "no action and no parameters",
			input: "/forgejo",
			want: output{
				"/forgejo",
				"",
				[]string(nil),
			},
		},
		{
			name:  "simple one-word label",
			input: `/forgejo subscribe DHaussermann/hello-world issues,label:"Help"`,
			want: output{
				"/forgejo",
				"subscribe",
				[]string{"DHaussermann/hello-world", `issues,label:"Help"`},
			},
		},
		{
			name:  "two-word label",
			input: `/forgejo subscribe DHaussermann/hello-world issues,label:"Help Wanted"`,
			want: output{
				"/forgejo",
				"subscribe",
				[]string{"DHaussermann/hello-world", `issues,label:"Help Wanted"`},
			},
		},
		{
			name:  "multi-word label",
			input: `/forgejo subscribe DHaussermann/hello-world issues,label:"Good First Issue"`,
			want: output{
				"/forgejo",
				"subscribe",
				[]string{"DHaussermann/hello-world", `issues,label:"Good First Issue"`},
			},
		},
		{
			name:  "multiple spaces inside double-quotes",
			input: `/forgejo subscribe DHaussermann/hello-world issues,label:"Help    Wanted"`,
			want: output{
				"/forgejo",
				"subscribe",
				[]string{"DHaussermann/hello-world", `issues,label:"Help    Wanted"`},
			},
		},
		{
			name:  "multiple spaces outside of double-quotes",
			input: `  /forgejo    subscribe     DHaussermann/hello-world issues,label:"Help Wanted"`,
			want: output{
				"/forgejo",
				"subscribe",
				[]string{"DHaussermann/hello-world", `issues,label:"Help Wanted"`},
			},
		},
		{
			name:  "trailing whitespaces",
			input: `/forgejo subscribe DHaussermann/hello-world issues,label:"Help Wanted" `,
			want: output{
				"/forgejo",
				"subscribe",
				[]string{"DHaussermann/hello-world", `issues,label:"Help Wanted"`},
			},
		},
		{
			name:  "non-ASCII characters",
			input: `/forgejo subscribe طماطم issues,label:"日本語"`,
			want: output{
				"/forgejo",
				"subscribe",
				[]string{"طماطم", `issues,label:"日本語"`},
			},
		},
		{
			name:  "line breaks",
			input: "/forgejo \nsubscribe\nDHaussermann/hello-world\nissues,label:\"Good First Issue\"",
			want: output{
				"/forgejo",
				"subscribe",
				[]string{"DHaussermann/hello-world", `issues,label:"Good First Issue"`},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			command, action, parameters := parseCommand(tc.input)
			got := output{command, action, parameters}
			testFailureMessage := fmt.Sprintf("validateFeatures() = %v, want %v", got, tc.want)
			assert.EqualValues(t, tc.want, got, testFailureMessage)
		})
	}
}

func TestCheckConflictingFeatures(t *testing.T) {
	type output struct {
		valid               bool
		conflictingFeatures []string
	}
	tests := []struct {
		name string
		args []string
		want output
	}{
		{
			name: "no conflicts",
			args: []string{"creates", "pushes", "issue_comments"},
			want: output{true, nil},
		},
		{
			name: "conflict with issue and issue creation",
			args: []string{"pulls", "issues", "issue_creations"},
			want: output{false, []string{"issues", "issue_creations"}},
		},
		{
			name: "conflict with pulls and pulls created",
			args: []string{"pulls", "issues", "pulls_created"},
			want: output{false, []string{"pulls", "pulls_created"}},
		},
		{
			name: "conflict with pulls and pulls merged",
			args: []string{"pulls", "pushes", "pulls_merged"},
			want: output{false, []string{"pulls", "pulls_merged"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, fs := checkFeatureConflict(tt.args)
			got := output{ok, fs}
			testFailureMessage := fmt.Sprintf("checkFeatureConflict() = %v, want %v", got, tt.want)
			assert.EqualValues(t, tt.want, got, testFailureMessage)
		})
	}
}
