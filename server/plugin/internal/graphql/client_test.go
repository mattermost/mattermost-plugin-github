package graphql

import (
	"flag"
	"testing"

	"github.com/mattermost/mattermost-plugin-github/server/plugin/internal/graphql/query"
	"golang.org/x/oauth2"
)

var userToken string

func init() {
	flag.StringVar(&userToken, "token", "", "Github user access token")
}

/**
 Following test creates the query below:

	query query {
	 search(last: 100, type: ISSUE, query: "author:srgyrn is:pr is:closed archived:false") {
	   issueCount
	   nodes {
		 ... on PullRequest {
		   id
		   number
		   body
		   state
		   url
		   title
		   mergeable
		   reviewDecision
		   reviewRequests(first: 10) {
			 totalCount
			 nodes {
			   requestedReviewer {
				 ... on User {
				   login
				 }
			   }
			 }
		   }
		   reviews(first: 10) {
			 nodes {
			   state
			 }
			 totalCount
		   }
		   repository {
			 name
			 nameWithOwner
		   }
		 }
	   }
	 }
	}
*/
func TestClient_ExecuteQuery(t *testing.T) {
	if userToken == "" {
		t.Skipf("empty user access token, skipping test")
	}

	var pru *query.Union
	var repo *query.Object

	tc, err := query.NewScalar("TotalCount", "Int")
	if err != nil {
		t.Errorf("error creating query: %v", err)
		return
	}

	// PullRequest union
	pru, err = query.NewUnion("PullRequest")
	if err != nil {
		t.Errorf("error creating query: %v", err)
		return
	}
	sg, err := query.NewScalarGroup(map[string]string{
		"id":             "id",
		"body":           "String",
		"number":         "Int",
		"reviewDecision": "PullRequestReviewDecision",
		"state":          "PullRequestState",
		"title":          "String",
		"url":            "URI",
		"closedAt":       "DateTime",
	})
	if err != nil {
		t.Errorf("error creating query: %v", err)
		return
	}
	pru.AddScalarGroup(sg)
	// PullRequest union

	// Repository object
	repo, err = query.NewObject("Repository")
	if err != nil {
		t.Errorf("error creating query: %v", err)
		return
	}
	sg, err = query.NewScalarGroup(map[string]string{
		"Name":          "String",
		"NameWithOwner": "String",
	})
	if err != nil {
		t.Errorf("error creating query: %v", err)
		return
	}
	repo.AddScalarGroup(sg)
	pru.AddObject(repo)
	// Repository object

	// Reviews object
	rv, err := query.NewObject("Reviews", query.SetFirst(10))
	if err != nil {
		t.Errorf("error creating query: %v", err)
		return
	}

	s, err := query.NewScalar("State", "PullRequestState")
	if err != nil {
		t.Errorf("error creating query: %v", err)
		return
	}

	rvnl := query.NewNodeList()
	rvnl.AddScalar(s)

	rv.AddScalar(tc)
	if err = rv.SetNodeList(rvnl); err != nil {
		t.Errorf("error creating query: %v", err)
		return
	}

	pru.AddObject(rv)
	// Reviews object

	// ReviewRequests object
	s, err = query.NewScalar("Login", "String")
	if err != nil {
		t.Errorf("error creating query: %v", err)
		return
	}
	// ReviewRequests object

	// RequestedReviewer object
	userUnion, err := query.NewUnion("User")
	if err != nil {
		t.Errorf("error creating query: %v", err)
		return
	}
	userUnion.AddScalar(s)

	reqRevr, _ := query.NewObject("RequestedReviewer")
	reqRevr.AddUnion(userUnion)
	// RequestedReviewer object

	// ReviewRequests object
	reqRev, _ := query.NewObject("ReviewRequests", query.SetFirst(10))
	reqRev.AddScalar(tc) // reuse of tc

	nl := query.NewNodeList()
	nl.AddObject(reqRevr)
	if err = reqRev.SetNodeList(nl); err != nil {
		t.Errorf("error creating query: %v", err)
		return
	}
	pru.AddObject(reqRev)
	// ReviewRequests object

	// Search object
	mainQuery, err := query.NewObject(
		"Search",
		query.SetFirst(100),
		query.SetQuery("author:srgyrn is:pr is:open archived:false"),
		query.SetSearchType("ISSUE"),
	)
	if err != nil {
		t.Errorf("error creating query: %v", err)
		return
	}

	s, err = query.NewScalar("IssueCount", "Int")
	if err != nil {
		t.Errorf("error creating query: %v", err)
		return
	}
	mainQuery.AddScalar(s)

	nl = query.NewNodeList()
	nl.AddUnion(pru)
	if err = mainQuery.SetNodeList(nl); err != nil {
		t.Errorf("error creating query: %v", err)
		return
	}

	client := NewClient(oauth2.Token{AccessToken: userToken}, "srgyrn", "", "")
	result, err := client.ExecuteQuery(mainQuery)
	if err != nil {
		t.Errorf("error creating query: %v", err)
		return
	}

	searchResult, err := result.Get("Search")
	if err != nil {
		t.Errorf("expected field not found in result: \n\n%#v", result)
		return
	}

	if !result.IsChildTypeResult("Search") {
		t.Errorf("expected Search to be type graphql.Result, got: %T", searchResult)
		return
	}

	if _, err := searchResult.(Response).Get("IssueCount"); err != nil {
		t.Errorf("expected field not found in result: \n\n%#v", result)
		return
	}

	if searchResult.(Response).GetChildType("Nodes") != TypeResponseSlice {
		t.Errorf("unexpected field type for result[\"Search\"][\"Nodes\"]")
		return
	}

	t.Logf("\n\nresult: %#v", result)
}
