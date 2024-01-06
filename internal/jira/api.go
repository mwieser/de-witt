package jira

type PostWorklogRequest struct {
	Started   string          `json:"started"`
	TimeSpent string          `json:"timeSpent"`
	Comment   *WorklogComment `json:"comment,omitempty"`
}

type WorklogComment struct {
	Version int                     `json:"version"`
	Type    string                  `json:"type"`
	Content []WorklogCommentWrapper `json:"content"`
}

type WorklogCommentWrapper struct {
	Content []CommentContent `json:"content"`
	Type    string           `json:"type"`
}

type CommentContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func buildWorklogComment(text string) *WorklogComment {
	return &WorklogComment{
		Version: 1,
		Type:    "doc",
		Content: []WorklogCommentWrapper{
			{
				Content: []CommentContent{
					{
						Type: "text",
						Text: text,
					},
				},
				Type: "paragraph",
			},
		},
	}
}
