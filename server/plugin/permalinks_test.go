package plugin

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-github/v48/github"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetReplacements(t *testing.T) {
	p := NewPlugin()

	tcs := []struct {
		name            string
		input           string
		numReplacements int
		replacements    []replacement
	}{
		{
			name:            "basic one link",
			input:           "start https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22 lorem ipsum",
			numReplacements: 1,
			replacements: []replacement{
				{
					index: 6,
					word:  "https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22",
					permalinkInfo: struct {
						haswww string
						commit string
						user   string
						repo   string
						path   string
						line   string
					}{
						haswww: "",
						commit: "cbb25838a61872b624ac512556d7bc932486a64c",
						line:   "L15-L22",
						path:   "app/authentication.go",
						user:   "mattermost",
						repo:   "mattermost-server",
					},
				},
			},
		}, {
			name:            "duplicate expansions",
			input:           "start https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22 lorem ipsum https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22 lorem ipsum",
			numReplacements: 2,
			replacements: []replacement{
				{
					index: 6,
					word:  "https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22",
					permalinkInfo: struct {
						haswww string
						commit string
						user   string
						repo   string
						path   string
						line   string
					}{
						commit: "cbb25838a61872b624ac512556d7bc932486a64c",
						haswww: "",
						line:   "L15-L22",
						path:   "app/authentication.go",
						user:   "mattermost",
						repo:   "mattermost-server",
					},
				}, {
					index: 142,
					word:  "https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22",
					permalinkInfo: struct {
						haswww string
						commit string
						user   string
						repo   string
						path   string
						line   string
					}{
						commit: "cbb25838a61872b624ac512556d7bc932486a64c",
						haswww: "",
						line:   "L15-L22",
						path:   "app/authentication.go",
						user:   "mattermost",
						repo:   "mattermost-server",
					},
				},
			},
		}, {
			name:            "inside link",
			input:           "should not expand [link](https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22) here",
			numReplacements: 0,
			replacements:    []replacement{},
		}, {
			name:            "one link, one expansion",
			input:           "first should not expand [link](https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22) this should https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22 lorem ipsum",
			numReplacements: 1,
			replacements: []replacement{
				{
					index: 168,
					word:  "https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22",
					permalinkInfo: struct {
						haswww string
						commit string
						user   string
						repo   string
						path   string
						line   string
					}{
						commit: "cbb25838a61872b624ac512556d7bc932486a64c",
						haswww: "",
						line:   "L15-L22",
						path:   "app/authentication.go",
						user:   "mattermost",
						repo:   "mattermost-server",
					},
				},
			},
		}, {
			name:            "one expansion, one link",
			input:           "first should expand https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22 lorem ipsum , this should not [link](https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22)",
			numReplacements: 1,
			replacements: []replacement{
				{
					index: 20,
					word:  "https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22",
					permalinkInfo: struct {
						haswww string
						commit string
						user   string
						repo   string
						path   string
						line   string
					}{
						commit: "cbb25838a61872b624ac512556d7bc932486a64c",
						haswww: "",
						line:   "L15-L22",
						path:   "app/authentication.go",
						user:   "mattermost",
						repo:   "mattermost-server",
					},
				},
			},
		}, {
			name:            "2 links",
			input:           "both should not expand- [link](https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22) and [link](https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22)",
			numReplacements: 0,
			replacements:    []replacement{},
		}, {
			name:            "multiple expansions",
			input:           "multiple - https://github.com/golang/go/blob/27fc32ff01cc699e160890546816bd99d6c57823/src/debug/macho/macho.go#L13-L16 second https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22",
			numReplacements: 2,
			replacements: []replacement{
				{
					index: 11,
					word:  "https://github.com/golang/go/blob/27fc32ff01cc699e160890546816bd99d6c57823/src/debug/macho/macho.go#L13-L16",
					permalinkInfo: struct {
						haswww string
						commit string
						user   string
						repo   string
						path   string
						line   string
					}{
						commit: "27fc32ff01cc699e160890546816bd99d6c57823",
						haswww: "",
						line:   "L13-L16",
						path:   "src/debug/macho/macho.go",
						user:   "golang",
						repo:   "go",
					},
				}, {
					index: 126,
					word:  "https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22",
					permalinkInfo: struct {
						haswww string
						commit string
						user   string
						repo   string
						path   string
						line   string
					}{
						commit: "cbb25838a61872b624ac512556d7bc932486a64c",
						haswww: "",
						line:   "L15-L22",
						path:   "app/authentication.go",
						user:   "mattermost",
						repo:   "mattermost-server",
					},
				},
			},
		}, {
			name:            "single line",
			input:           "this is a one line permalink https://github.com/mattermost/mattermost-server/blob/4225977966cf0855c8a5e55f8a0fef702b19dc18/api4/bot.go#L16",
			numReplacements: 1,
			replacements: []replacement{
				{
					index: 29,
					word:  "https://github.com/mattermost/mattermost-server/blob/4225977966cf0855c8a5e55f8a0fef702b19dc18/api4/bot.go#L16",
					permalinkInfo: struct {
						haswww string
						commit string
						user   string
						repo   string
						path   string
						line   string
					}{
						commit: "4225977966cf0855c8a5e55f8a0fef702b19dc18",
						haswww: "",
						line:   "L16",
						path:   "api4/bot.go",
						user:   "mattermost",
						repo:   "mattermost-server",
					},
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			replacements := p.getReplacements(tc.input)
			require.Equalf(t, tc.numReplacements, len(replacements), "unexpected number of replacements for %s", tc.input)
			for i, r := range replacements {
				assert.Equalf(t, tc.replacements[i].index, r.index, "unexpected replacement index")
				assert.Equalf(t, tc.replacements[i].word, r.word, "unexpected replacement word")
				assert.Equalf(t, tc.replacements[i].permalinkInfo.commit, r.permalinkInfo.commit, "unexpected github commit")
				assert.Equalf(t, tc.replacements[i].permalinkInfo.haswww, r.permalinkInfo.haswww, "unexpected github www domain")
				assert.Equalf(t, tc.replacements[i].permalinkInfo.line, r.permalinkInfo.line, "unexpected line number")
				assert.Equalf(t, tc.replacements[i].permalinkInfo.path, r.permalinkInfo.path, "unexpected file path")
				assert.Equalf(t, tc.replacements[i].permalinkInfo.user, r.permalinkInfo.user, "unexpected github user")
				assert.Equalf(t, tc.replacements[i].permalinkInfo.repo, r.permalinkInfo.repo, "unexpected github repo")
			}
		})
	}
}

func TestMakeReplacements(t *testing.T) {
	p := NewPlugin()
	mockPluginAPI := &plugintest.API{}
	mockPluginAPI.On("LogError", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockPluginAPI.On("LogWarn", mock.Anything, mock.Anything)
	p.SetAPI(mockPluginAPI)

	tcs := []struct {
		name         string
		input        string
		output       string
		replacements []replacement
	}{
		{
			name:   "basic one link",
			input:  "start https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22 lorem ipsum",
			output: "start \n[mattermost/mattermost-server/app/authentication.go](https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22)\n```go\ntype TokenLocation int\n\nconst (\n\tTokenLocationNotFound TokenLocation = iota\n\tTokenLocationHeader\n\tTokenLocationCookie\n\tTokenLocationQueryString\n)\n```\n lorem ipsum",
			replacements: []replacement{
				{
					index: 6,
					word:  "https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22",
					permalinkInfo: struct {
						haswww string
						commit string
						user   string
						repo   string
						path   string
						line   string
					}{
						haswww: "",
						commit: "cbb25838a61872b624ac512556d7bc932486a64c",
						line:   "L15-L22",
						path:   "app/authentication.go",
						user:   "mattermost",
						repo:   "mattermost-server",
					},
				},
			},
		},
		{
			name:   "duplicate expansions",
			input:  "start https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22 lorem ipsum https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22 lorem ipsum",
			output: "start \n[mattermost/mattermost-server/app/authentication.go](https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22)\n```go\ntype TokenLocation int\n\nconst (\n\tTokenLocationNotFound TokenLocation = iota\n\tTokenLocationHeader\n\tTokenLocationCookie\n\tTokenLocationQueryString\n)\n```\n lorem ipsum \n[mattermost/mattermost-server/app/authentication.go](https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22)\n```go\ntype TokenLocation int\n\nconst (\n\tTokenLocationNotFound TokenLocation = iota\n\tTokenLocationHeader\n\tTokenLocationCookie\n\tTokenLocationQueryString\n)\n```\n lorem ipsum",
			replacements: []replacement{
				{
					index: 6,
					word:  "https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22",
					permalinkInfo: struct {
						haswww string
						commit string
						user   string
						repo   string
						path   string
						line   string
					}{
						commit: "cbb25838a61872b624ac512556d7bc932486a64c",
						haswww: "",
						line:   "L15-L22",
						path:   "app/authentication.go",
						user:   "mattermost",
						repo:   "mattermost-server",
					},
				}, {
					index: 142,
					word:  "https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L15-L22",
					permalinkInfo: struct {
						haswww string
						commit string
						user   string
						repo   string
						path   string
						line   string
					}{
						commit: "cbb25838a61872b624ac512556d7bc932486a64c",
						haswww: "",
						line:   "L15-L22",
						path:   "app/authentication.go",
						user:   "mattermost",
						repo:   "mattermost-server",
					},
				},
			},
		},
		{
			name:   "bad commit hash",
			input:  "start https://github.com/mattermost/mattermost-server/blob/badhash/app/authentication.go#L15-L22 lorem ipsum",
			output: "start https://github.com/mattermost/mattermost-server/blob/badhash/app/authentication.go#L15-L22 lorem ipsum",
			replacements: []replacement{
				{
					index: 6,
					word:  "https://github.com/mattermost/mattermost-server/blob/badhash/app/authentication.go#L15-L22",
					permalinkInfo: struct {
						haswww string
						commit string
						user   string
						repo   string
						path   string
						line   string
					}{
						haswww: "",
						commit: "badhash",
						line:   "L15-L22",
						path:   "app/authentication.go",
						user:   "mattermost",
						repo:   "mattermost-server",
					},
				},
			},
		},
		{
			name:   "bad line range",
			input:  "start https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L22-L15 lorem ipsum",
			output: "start https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L22-L15 lorem ipsum",
			replacements: []replacement{
				{
					index: 6,
					word:  "https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go#L22-L15",
					permalinkInfo: struct {
						haswww string
						commit string
						user   string
						repo   string
						path   string
						line   string
					}{
						haswww: "",
						commit: "cbb25838a61872b624ac512556d7bc932486a64c",
						line:   "L22-L15",
						path:   "app/authentication.go",
						user:   "mattermost",
						repo:   "mattermost-server",
					},
				},
			},
		},
		{
			name:   "bad file content",
			input:  "start https://github.com/badorg/badrepo/path/file.go#L1-L2 lorem ipsum",
			output: "start https://github.com/badorg/badrepo/path/file.go#L1-L2 lorem ipsum",
			replacements: []replacement{
				{
					index: 5,
					word:  "https://github.com/badorg/badrepo/path/file.go#L1-L2",
					permalinkInfo: struct {
						haswww string
						commit string
						user   string
						repo   string
						path   string
						line   string
					}{
						haswww: "",
						commit: "cbb25838a61872b624ac512556d7bc932486a64c",
						line:   "L1-L2",
						path:   "path/file.go",
						user:   "badorg",
						repo:   "badrepo",
					},
				},
			},
		},
	}
	client, close := getClient()
	defer close()

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			msg := p.makeReplacements(tc.input, tc.replacements, client)
			assert.Equalf(t, tc.output, msg, "mismatched output")
		})
	}

	mockPluginAPI.AssertCalled(t, "LogError", "Bad git commit hash in permalink", "error", "encoding/hex: invalid byte: U+0068 'h'", "hash", "badhash")
	mockPluginAPI.AssertCalled(t, "LogError", "Error while fetching file contents", "error", "unmarshalling failed for both file and directory content: unexpected end of JSON input and unexpected end of JSON input", "path", "path/file.go")
}

const (
	baseURLPath = "/api-v3"
)

func getClient() (*github.Client, func()) {
	apiHandler := http.NewServeMux()
	apiHandler.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/api-v3/repos/mattermost/mattermost-server/contents/app/authentication.go":
			fmt.Fprintln(w, `{
  "name": "authentication.go",
  "path": "app/authentication.go",
  "sha": "c5c4ebf9077d04306ce8eca1e451421e4df7ca3c",
  "size": 7950,
  "url": "https://api.github.com/repos/mattermost/mattermost-server/contents/app/authentication.go?ref=cbb25838a61872b624ac512556d7bc932486a64c",
  "html_url": "https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go",
  "git_url": "https://api.github.com/repos/mattermost/mattermost-server/git/blobs/c5c4ebf9077d04306ce8eca1e451421e4df7ca3c",
  "download_url": "https://raw.githubusercontent.com/mattermost/mattermost-server/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go",
  "type": "file",
  "content": "Ly8gQ29weXJpZ2h0IChjKSAyMDE2LXByZXNlbnQgTWF0dGVybW9zdCwgSW5j\nLiBBbGwgUmlnaHRzIFJlc2VydmVkLgovLyBTZWUgTGljZW5zZS50eHQgZm9y\nIGxpY2Vuc2UgaW5mb3JtYXRpb24uCgpwYWNrYWdlIGFwcAoKaW1wb3J0ICgK\nCSJuZXQvaHR0cCIKCSJzdHJpbmdzIgoKCSJnaXRodWIuY29tL21hdHRlcm1v\nc3QvbWF0dGVybW9zdC1zZXJ2ZXIvbW9kZWwiCgkiZ2l0aHViLmNvbS9tYXR0\nZXJtb3N0L21hdHRlcm1vc3Qtc2VydmVyL3NlcnZpY2VzL21mYSIKCSJnaXRo\ndWIuY29tL21hdHRlcm1vc3QvbWF0dGVybW9zdC1zZXJ2ZXIvdXRpbHMiCikK\nCnR5cGUgVG9rZW5Mb2NhdGlvbiBpbnQKCmNvbnN0ICgKCVRva2VuTG9jYXRp\nb25Ob3RGb3VuZCBUb2tlbkxvY2F0aW9uID0gaW90YQoJVG9rZW5Mb2NhdGlv\nbkhlYWRlcgoJVG9rZW5Mb2NhdGlvbkNvb2tpZQoJVG9rZW5Mb2NhdGlvblF1\nZXJ5U3RyaW5nCikKCmZ1bmMgKHRsIFRva2VuTG9jYXRpb24pIFN0cmluZygp\nIHN0cmluZyB7Cglzd2l0Y2ggdGwgewoJY2FzZSBUb2tlbkxvY2F0aW9uTm90\nRm91bmQ6CgkJcmV0dXJuICJOb3QgRm91bmQiCgljYXNlIFRva2VuTG9jYXRp\nb25IZWFkZXI6CgkJcmV0dXJuICJIZWFkZXIiCgljYXNlIFRva2VuTG9jYXRp\nb25Db29raWU6CgkJcmV0dXJuICJDb29raWUiCgljYXNlIFRva2VuTG9jYXRp\nb25RdWVyeVN0cmluZzoKCQlyZXR1cm4gIlF1ZXJ5U3RyaW5nIgoJZGVmYXVs\ndDoKCQlyZXR1cm4gIlVua25vd24iCgl9Cn0KCmZ1bmMgKGEgKkFwcCkgSXNQ\nYXNzd29yZFZhbGlkKHBhc3N3b3JkIHN0cmluZykgKm1vZGVsLkFwcEVycm9y\nIHsKCglpZiAqYS5Db25maWcoKS5TZXJ2aWNlU2V0dGluZ3MuRW5hYmxlRGV2\nZWxvcGVyIHsKCQlyZXR1cm4gbmlsCgl9CgoJcmV0dXJuIHV0aWxzLklzUGFz\nc3dvcmRWYWxpZFdpdGhTZXR0aW5ncyhwYXNzd29yZCwgJmEuQ29uZmlnKCku\nUGFzc3dvcmRTZXR0aW5ncykKfQoKZnVuYyAoYSAqQXBwKSBDaGVja1Bhc3N3\nb3JkQW5kQWxsQ3JpdGVyaWEodXNlciAqbW9kZWwuVXNlciwgcGFzc3dvcmQg\nc3RyaW5nLCBtZmFUb2tlbiBzdHJpbmcpICptb2RlbC5BcHBFcnJvciB7Cglp\nZiBlcnIgOj0gYS5DaGVja1VzZXJQcmVmbGlnaHRBdXRoZW50aWNhdGlvbkNy\naXRlcmlhKHVzZXIsIG1mYVRva2VuKTsgZXJyICE9IG5pbCB7CgkJcmV0dXJu\nIGVycgoJfQoKCWlmIGVyciA6PSBhLmNoZWNrVXNlclBhc3N3b3JkKHVzZXIs\nIHBhc3N3b3JkKTsgZXJyICE9IG5pbCB7CgkJaWYgcGFzc0VyciA6PSBhLlNy\ndi5TdG9yZS5Vc2VyKCkuVXBkYXRlRmFpbGVkUGFzc3dvcmRBdHRlbXB0cyh1\nc2VyLklkLCB1c2VyLkZhaWxlZEF0dGVtcHRzKzEpOyBwYXNzRXJyICE9IG5p\nbCB7CgkJCXJldHVybiBwYXNzRXJyCgkJfQoJCXJldHVybiBlcnIKCX0KCglp\nZiBlcnIgOj0gYS5DaGVja1VzZXJNZmEodXNlciwgbWZhVG9rZW4pOyBlcnIg\nIT0gbmlsIHsKCQkvLyBJZiB0aGUgbWZhVG9rZW4gaXMgbm90IHNldCwgd2Ug\nYXNzdW1lIHRoZSBjbGllbnQgdXNlZCB0aGlzIGFzIGEgcHJlLWZsaWdodCBy\nZXF1ZXN0IHRvIHF1ZXJ5IHRoZSBzZXJ2ZXIKCQkvLyBhYm91dCB0aGUgTUZB\nIHN0YXRlIG9mIHRoZSB1c2VyIGluIHF1ZXN0aW9uCgkJaWYgbWZhVG9rZW4g\nIT0gIiIgewoJCQlpZiBwYXNzRXJyIDo9IGEuU3J2LlN0b3JlLlVzZXIoKS5V\ncGRhdGVGYWlsZWRQYXNzd29yZEF0dGVtcHRzKHVzZXIuSWQsIHVzZXIuRmFp\nbGVkQXR0ZW1wdHMrMSk7IHBhc3NFcnIgIT0gbmlsIHsKCQkJCXJldHVybiBw\nYXNzRXJyCgkJCX0KCQl9CgkJcmV0dXJuIGVycgoJfQoKCWlmIHBhc3NFcnIg\nOj0gYS5TcnYuU3RvcmUuVXNlcigpLlVwZGF0ZUZhaWxlZFBhc3N3b3JkQXR0\nZW1wdHModXNlci5JZCwgMCk7IHBhc3NFcnIgIT0gbmlsIHsKCQlyZXR1cm4g\ncGFzc0VycgoJfQoKCWlmIGVyciA6PSBhLkNoZWNrVXNlclBvc3RmbGlnaHRB\ndXRoZW50aWNhdGlvbkNyaXRlcmlhKHVzZXIpOyBlcnIgIT0gbmlsIHsKCQly\nZXR1cm4gZXJyCgl9CgoJcmV0dXJuIG5pbAp9CgovLyBUaGlzIHRvIGJlIHVz\nZWQgZm9yIHBsYWNlcyB3ZSBjaGVjayB0aGUgdXNlcnMgcGFzc3dvcmQgd2hl\nbiB0aGV5IGFyZSBhbHJlYWR5IGxvZ2dlZCBpbgpmdW5jIChhICpBcHApIERv\ndWJsZUNoZWNrUGFzc3dvcmQodXNlciAqbW9kZWwuVXNlciwgcGFzc3dvcmQg\nc3RyaW5nKSAqbW9kZWwuQXBwRXJyb3IgewoJaWYgZXJyIDo9IGNoZWNrVXNl\nckxvZ2luQXR0ZW1wdHModXNlciwgKmEuQ29uZmlnKCkuU2VydmljZVNldHRp\nbmdzLk1heGltdW1Mb2dpbkF0dGVtcHRzKTsgZXJyICE9IG5pbCB7CgkJcmV0\ndXJuIGVycgoJfQoKCWlmIGVyciA6PSBhLmNoZWNrVXNlclBhc3N3b3JkKHVz\nZXIsIHBhc3N3b3JkKTsgZXJyICE9IG5pbCB7CgkJaWYgcGFzc0VyciA6PSBh\nLlNydi5TdG9yZS5Vc2VyKCkuVXBkYXRlRmFpbGVkUGFzc3dvcmRBdHRlbXB0\ncyh1c2VyLklkLCB1c2VyLkZhaWxlZEF0dGVtcHRzKzEpOyBwYXNzRXJyICE9\nIG5pbCB7CgkJCXJldHVybiBwYXNzRXJyCgkJfQoJCXJldHVybiBlcnIKCX0K\nCglpZiBwYXNzRXJyIDo9IGEuU3J2LlN0b3JlLlVzZXIoKS5VcGRhdGVGYWls\nZWRQYXNzd29yZEF0dGVtcHRzKHVzZXIuSWQsIDApOyBwYXNzRXJyICE9IG5p\nbCB7CgkJcmV0dXJuIHBhc3NFcnIKCX0KCglyZXR1cm4gbmlsCn0KCmZ1bmMg\n",
  "encoding": "base64",
  "_links": {
    "self": "https://api.github.com/repos/mattermost/mattermost-server/contents/app/authentication.go?ref=cbb25838a61872b624ac512556d7bc932486a64c",
    "git": "https://api.github.com/repos/mattermost/mattermost-server/git/blobs/c5c4ebf9077d04306ce8eca1e451421e4df7ca3c",
    "html": "https://github.com/mattermost/mattermost-server/blob/cbb25838a61872b624ac512556d7bc932486a64c/app/authentication.go"
  }
}`)
		case "/api-v3/repos/badorg/badrepo/path/file.go":
			fmt.Fprintln(w, `{
  "sha": "c5c4ebf9077d04306ce8eca1e451421e4df7ca3c",
  "size": 7950,
  "url": "https://api.github.com/repos/badorg/badrepo/path/file.g",
  "html_url": "https://github.com/badorg/badrepo/path/file.g",
  "git_url": "https://api.github.com/repos/badorg/badrepo/path/file.g",
  "download_url": "https://raw.githubusercontent.com/badorg/badrepo/path/file.g",
  "type": "file",
  "content": "badinput",
  "encoding": "base64"
}`)
		}
	})

	// server is a test HTTP server used to provide mock API responses.
	server := httptest.NewServer(apiHandler)

	// client is the GitHub client being tested and is
	// configured to use test server.
	client := github.NewClient(nil)
	url, _ := url.Parse(server.URL + baseURLPath + "/")
	client.BaseURL = url
	client.UploadURL = url
	return client, server.Close
}
