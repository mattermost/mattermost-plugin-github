package main

import "fmt"

func getMentionSearchQuery(username, org string) string {
	return bulidSearchQuery("is:open mentions:%v archived:false %v", username, org)
}

func getReviewSearchQuery(username, org string) string {
	return bulidSearchQuery("is:pr is:open review-requested:%v archived:false %v", username, org)
}

func bulidSearchQuery(query, username, org string) string {
	orgField := ""
	if len(org) != 0 {
		orgField = fmt.Sprintf("org:%v", org)
	}

	return fmt.Sprintf(query, username, orgField)
}
