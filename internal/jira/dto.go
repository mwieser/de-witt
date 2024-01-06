package jira

import "time"

type WorklogRecord struct {
	JiraURL          string
	WorklogID        string
	IssueKey         string
	TimeSpent        string
	TimeSpentSeconds int
	ProjectKey       string
	ParrentKey       string
	Started          time.Time
	Comment          string
}
