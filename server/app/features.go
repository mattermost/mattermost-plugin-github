package app

import "strings"

const (
	featureIssueCreation = "issue_creations"
	featureIssues        = "issues"
	featurePullsMerged   = "pulls_merged"
	featurePushes        = "pushes"
	featureCreates       = "creates"
	featureDeletes       = "deletes"
	featureIssueComments = "issue_comments"
	featurePullReviews   = "pull_reviews"
	featurePulls         = "pulls"
	featureStars         = "stars"
)

var validFeatures = map[string]bool{
	featureIssueCreation: true,
	featureIssues:        true,
	featurePulls:         true,
	featurePullsMerged:   true,
	featurePushes:        true,
	featureCreates:       true,
	featureDeletes:       true,
	featureIssueComments: true,
	featurePullReviews:   true,
	featureStars:         true,
}

// ValidateFeatures returns false when 1 or more given features
// are invalid along with a list of the invalid features.
func ValidateFeatures(features []string) (bool, []string) {
	valid := true
	invalidFeatures := []string{}
	hasLabel := false
	for _, f := range features {
		if _, ok := validFeatures[f]; ok {
			continue
		}
		if strings.HasPrefix(f, "label") {
			hasLabel = true
			continue
		}
		invalidFeatures = append(invalidFeatures, f)
		valid = false
	}
	if valid && hasLabel {
		// must have "pulls" or "issues" in features when using a label
		for _, f := range features {
			if f == featurePulls || f == featureIssues {
				return valid, invalidFeatures
			}
		}
		valid = false
	}
	return valid, invalidFeatures
}
