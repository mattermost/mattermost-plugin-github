package search

import (
	"github.com/mattermost/mattermost-plugin-github/server/plugin/internal/graphql/query"
)

func getPrsSearchQuery(username string) (*query.Object, error) {
	tc, err := query.NewScalar("TotalCount", "Int")
	if err != nil {
		return nil, err
	}

	userScalars, err := query.NewScalarGroup(map[string]string{
		"Login":      "String",
		"AvatarURL":  "URI",
		"URL":        "URI",
		"Name":       "String",
		"Email":      "String",
		"DatabaseID": "Int",
	})
	if err != nil {
		return nil, err
	}

	// User union
	userUnion, err := query.NewUnion("User")
	if err != nil {
		return nil, err
	}
	userUnion.AddScalarGroup(userScalars)

	// PullRequest union
	pru, err := query.NewUnion("PullRequest")
	if err != nil {
		return nil, err
	}
	pruScalars, err := query.NewScalarGroup(map[string]string{
		"ID":             "ID",
		"Body":           "String",
		"Number":         "Int",
		"ReviewDecision": "PullRequestReviewDecision",
		"State":          "PullRequestState",
		"Title":          "String",
		"URL":            "URI",
		"Mergeable":      "MergeableState",
	})
	if err != nil {
		return nil, err
	}
	pru.AddScalarGroup(pruScalars)
	// PullRequest union

	// Reviews object
	rv, err := query.NewObject("Reviews", query.SetFirst(10))
	if err != nil {
		return nil, err
	}

	// Author object
	author, err := query.NewObject("Author")
	if err != nil {
		return nil, err
	}
	author.AddUnion(userUnion)

	reviewScalars, err := query.NewScalarGroup(map[string]string{
		"State":      "PullRequestState",
		"Body":       "String",
		"URL":        "URI",
		"ID":         "ID",
		"DatabaseID": "Int",
	})
	if err != nil {
		return nil, err
	}

	rvnl := query.NewNodeList()
	rvnl.AddScalarGroup(reviewScalars)
	rvnl.AddObject(author)

	rv.AddScalar(tc)
	if err = rv.SetNodeList(rvnl); err != nil {
		return nil, err
	}

	pru.AddObject(rv)
	// Reviews object

	// ReviewRequests object
	reqRev, _ := query.NewObject("ReviewRequests", query.SetFirst(10))
	reqRev.AddScalar(tc)

	// RequestedReviewer object
	reqRevr, _ := query.NewObject("RequestedReviewer")
	reqRevr.AddUnion(userUnion)

	nl := query.NewNodeList()
	nl.AddObject(reqRevr)
	if err = reqRev.SetNodeList(nl); err != nil {
		return nil, err
	}
	pru.AddObject(reqRev)
	// ReviewRequests object

	// Search object
	mainQuery, err := query.NewObject(
		"Search",
		query.SetFirst(100),
		query.SetQuery("is:pr is:open author:"+username+" archived:false"),
		query.SetSearchType("ISSUE"),
	)
	if err != nil {
		return nil, err
	}

	issueCount, err := query.NewScalar("IssueCount", "Int")
	if err != nil {
		return nil, err
	}
	mainQuery.AddScalar(issueCount)

	nl = query.NewNodeList()
	nl.AddUnion(pru)
	if err = mainQuery.SetNodeList(nl); err != nil {
		return nil, err
	}

	return mainQuery, nil
}
