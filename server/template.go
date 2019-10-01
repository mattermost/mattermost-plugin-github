package main

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/pkg/errors"
)

var masterTemplate *template.Template
var gitHubToUsernameMappingCallback func(string) string

func init() {
	var funcMap = sprig.TxtFuncMap()

	// Try to parse out email footer junk
	funcMap["trimBody"] = func(body string) string {
		if strings.Contains(body, "notifications@github.com") {
			return strings.Split(body, "\n\nOn")[0]
		}

		return body
	}

	// Trim a ref to use in constructing a link.
	funcMap["trimRef"] = func(ref string) string {
		return strings.Replace(ref, "refs/heads/", "", 1)
	}

	// Resolve a GitHub username to the corresponding Mattermost username, if linked.
	funcMap["lookupMattermostUsername"] = func(githubUsername string) string {
		if gitHubToUsernameMappingCallback == nil {
			return ""
		}

		return gitHubToUsernameMappingCallback(githubUsername)
	}

	masterTemplate = template.Must(template.New("master").Funcs(funcMap).Parse(""))

	// The user template links to the corresponding GitHub user. If the GitHub user is a known
	// Mattermost user, their Mattermost handle is referenced as an at-mention instead.
	template.Must(masterTemplate.New("user").Parse(`
{{- $mattermostUsername := .GetLogin | lookupMattermostUsername}}
{{- if $mattermostUsername }}@{{$mattermostUsername}}
{{- else}}[{{.GetLogin}}]({{.GetHTMLURL}})
{{- end -}}
	`))

	// The repo template links to the corresponding repository.
	template.Must(masterTemplate.New("repo").Parse(
		`[\[{{.GetFullName}}\]]({{.GetHTMLURL}})`,
	))

	// The eventRepoPullRequest links to the corresponding pull request, anchored at the repo.
	template.Must(masterTemplate.New("eventRepoPullRequest").Parse(
		`[{{.GetRepo.GetFullName}}#{{.GetPullRequest.GetNumber}}]({{.GetPullRequest.GetHTMLURL}})`,
	))

	// The pullRequest links to the corresponding pull request, skipping the repo title.
	template.Must(masterTemplate.New("pullRequest").Parse(
		`[#{{.GetNumber}} {{.GetTitle}}]({{.GetHTMLURL}})`,
	))

	// The issue links to the corresponding issue.
	template.Must(masterTemplate.New("issue").Parse(
		`[#{{.GetNumber}} {{.GetTitle}}]({{.GetHTMLURL}})`,
	))

	// The eventRepoIssue links to the corresponding issue. Note that, for some events, the
	// issue *is* a pull request, and so we still use .GetIssue and this template accordingly.
	template.Must(masterTemplate.New("eventRepoIssue").Parse(
		`[{{.GetRepo.GetFullName}}#{{.GetIssue.GetNumber}}]({{.GetIssue.GetHTMLURL}})`,
	))

	template.Must(masterTemplate.New("newPR").Funcs(funcMap).Parse(`
#### {{.GetPullRequest.GetTitle}}
##### {{template "eventRepoPullRequest" .}}
#new-pull-request by {{template "user" .GetSender}} on [{{.GetPullRequest.GetCreatedAt.String}}]({{.GetPullRequest.GetHTMLURL}})

{{.GetPullRequest.GetBody}}
`))

	template.Must(masterTemplate.New("closedPR").Funcs(funcMap).Parse(`
{{template "repo" .GetRepo}} Pull request {{template "pullRequest" .GetPullRequest}} was
{{- if .GetPullRequest.GetMerged }} merged
{{- else }} closed
{{- end }} by {{template "user" .GetSender}}.
`))

	template.Must(masterTemplate.New("pullRequestLabelled").Funcs(funcMap).Parse(`
#### {{.GetPullRequest.GetTitle}}
##### {{template "eventRepoPullRequest" .}}
#pull-request-labeled ` + "`{{.GetLabel.GetName}}`" + ` by {{template "user" .GetSender}} on [{{.GetPullRequest.GetUpdatedAt.String}}]({{.GetPullRequest.GetHTMLURL}})
`))

	template.Must(masterTemplate.New("newIssue").Funcs(funcMap).Parse(`
#### {{.GetIssue.GetTitle}}
##### {{template "eventRepoIssue" .}}
#new-issue by {{template "user" .GetSender}} on [{{.GetIssue.GetCreatedAt.String}}]({{.GetIssue.GetHTMLURL}})

{{.GetIssue.GetBody}}
`))

	template.Must(masterTemplate.New("closedIssue").Funcs(funcMap).Parse(`
{{template "repo" .GetRepo}} Issue {{template "issue" .GetIssue}} closed by {{template "user" .GetSender}}.
`))

	template.Must(masterTemplate.New("issueLabelled").Funcs(funcMap).Parse(`
#### {{.GetIssue.GetTitle}}
##### {{template "eventRepoIssue" .}}
#issue-labeled ` + "`{{.GetLabel.GetName}}`" + ` by {{template "user" .GetSender}} on [{{.GetIssue.GetUpdatedAt.String}}]({{.GetIssue.GetHTMLURL}}).
`))

	template.Must(masterTemplate.New("pushedCommits").Funcs(funcMap).Parse(`
{{template "user" .GetSender}} {{if .GetForced}}force-{{end}}pushed [{{len .Commits}} new commit{{if ne (len .Commits) 1}}s{{end}}]({{.GetCompare}}) to [\[{{.GetRepo.GetFullName}}:{{.GetRef | trimRef}}\]]({{.GetRepo.GetHTMLURL}}/tree/{{.GetRef | trimRef}}):
{{range .Commits -}}
[` + "`{{.GetID | substr 0 6}}`" + `]({{.GetURL}}) {{.GetMessage}} - {{.GetCommitter.GetName}}
{{end -}}
`))

	template.Must(masterTemplate.New("newCreateMessage").Funcs(funcMap).Parse(`
{{template "user" .GetSender}} just created {{.GetRefType}} [\[{{.GetRepo.GetFullName}}:{{.GetRef}}\]]({{.GetRepo.GetHTMLURL}}/tree/{{.GetRef}})
`))

	template.Must(masterTemplate.New("newDeleteMessage").Funcs(funcMap).Parse(`
{{template "user" .GetSender}} just deleted {{.GetRefType}} \[{{.GetRepo.GetFullName}}:{{.GetRef}}]
`))

	template.Must(masterTemplate.New("issueComment").Funcs(funcMap).Parse(`
{{template "repo" .GetRepo}} New comment by {{template "user" .GetSender}} on {{template "issue" .Issue}}:

{{.GetComment.GetBody | trimBody}}
`))

	template.Must(masterTemplate.New("pullRequestReviewEvent").Funcs(funcMap).Parse(`
{{template "repo" .GetRepo}} {{template "user" .GetSender}}
{{- if eq .GetReview.GetState "APPROVED"}} approved
{{- else if eq .GetReview.GetState "COMMENTED"}} commented on
{{- else if eq .GetReview.GetState "CHANGES_REQUESTED"}} requested changes on
{{- end }} {{template "pullRequest" .GetPullRequest}}:

{{.Review.GetBody}}
`))

	template.Must(masterTemplate.New("newReviewComment").Funcs(funcMap).Parse(`
{{template "repo" .GetRepo}} New review comment by {{template "user" .GetSender}} on {{template "pullRequest" .GetPullRequest}}:

{{.GetComment.GetDiffHunk}}
{{.GetComment.GetBody | trimBody}}
`))

	template.Must(masterTemplate.New("commentMentionNotification").Funcs(funcMap).Parse(`
{{template "user" .GetSender}} mentioned you on [#{{.Issue.GetNumber}} {{.Issue.GetTitle}}]({{.GetComment.GetHTMLURL}}):
>{{.GetComment.GetBody | trimBody}}
`))

	template.Must(masterTemplate.New("commentAuthorPullRequestNotification").Funcs(funcMap).Parse(`
{{template "user" .GetSender}} commented on your pull request {{template "issue" .GetIssue}}
`))

	template.Must(masterTemplate.New("commentAuthorIssueNotification").Funcs(funcMap).Parse(`
{{template "user" .GetSender}} commented on your issue {{template "issue" .GetIssue}}
`))

	template.Must(masterTemplate.New("pullRequestNotification").Funcs(funcMap).Parse(`
{{template "user" .GetSender}}
{{- if eq .GetAction "review_requested" }} requested your review on
{{- else if eq .GetAction "closed" }}
    {{- if .GetPullRequest.GetMerged }} merged your pull request
    {{- else }} closed your pull request
    {{- end }}
{{- else if eq .GetAction "reopened" }} reopened your pull request
{{- else if eq .GetAction "assigned" }} assigned you to pull request
{{- end }} {{template "pullRequest" .GetPullRequest}}
`))

	template.Must(masterTemplate.New("issueNotification").Funcs(funcMap).Parse(`
{{template "user" .GetSender}}
{{- if eq .GetAction "closed" }} closed your issue
{{- else if eq .GetAction "reopened" }} reopened your issue
{{- else if eq .GetAction "assigned" }} assigned you to issue
{{- end }} {{template "issue" .GetIssue}}
`))

	template.Must(masterTemplate.New("pullRequestReviewNotification").Funcs(funcMap).Parse(`
{{template "user" .GetSender}}
{{- if eq .GetReview.GetState "approved" }} approved your pull request
{{- else if eq .GetReview.GetState "changes_requested" }} requested changes on your pull request
{{- else if eq .GetReview.GetState "commented" }} commented on your pull request
{{- end }} {{template "pullRequest" .GetPullRequest}}
`))
}

func registerGitHubToUsernameMappingCallback(callback func(string) string) {
	gitHubToUsernameMappingCallback = callback
}

func renderTemplate(name string, data interface{}) (string, error) {
	var output bytes.Buffer
	t := masterTemplate.Lookup(name)
	if t == nil {
		return "", errors.Errorf("no template named %s", name)
	}

	err := t.Execute(&output, data)
	if err != nil {
		return "", err
	}

	return output.String(), nil
}
