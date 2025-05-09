package jira

type DateTime struct {
	ISO8601  string `json:"iso8601"`
	Jira     string `json:"jira"`
	Friendly string `json:"friendly"`
}

type User struct {
	AccountID    string `json:"accountId"`
	EmailAddress string `json:"emailAddress"`
	DisplayName  string `json:"displayName"`
	Active       bool   `json:"active"`
	TimeZone     string `json:"timeZone"`
	Links        struct {
		JiraRest   string `json:"jiraRest"`
		AvatarUrls struct {
			Size16 string `json:"16x16"`
			Size32 string `json:"32x32"`
		} `json:"avatarUrls"`
	} `json:"_links"`
}

type RequestFieldValue struct {
	FieldID string      `json:"fieldId"`
	Label   string      `json:"label"`
	Value   interface{} `json:"value"`
}

type Status struct {
	Status         string   `json:"status"`
	StatusCategory string   `json:"statusCategory"`
	StatusDate     DateTime `json:"statusDate"`
}

type Comment struct {
	ID        string   `json:"id"`
	Body      string   `json:"body"`
	Created   DateTime `json:"created"`
	Updated   DateTime `json:"updated"`
	Author    User     `json:"author"`
	JSDPublic bool     `json:"jsdPublic"`
}

type CommentsResponse struct {
	Start    int       `json:"start"`
	Limit    int       `json:"limit"`
	Size     int       `json:"size"`
	Comments []Comment `json:"values"`
}

type Issue struct {
	Expands       []string            `json:"_expands"`
	IssueID       string              `json:"issueId"`
	IssueKey      string              `json:"issueKey"`
	Summary       string              `json:"summary"`
	RequestTypeID string              `json:"requestTypeId"`
	ServiceDeskID string              `json:"serviceDeskId"`
	CreatedDate   DateTime            `json:"createdDate"`
	Reporter      User                `json:"reporter"`
	RequestFields []RequestFieldValue `json:"requestFieldValues"`
	CurrentStatus Status              `json:"currentStatus"`
	Comments      []Comment           `json:"comments,omitempty"`
	Links         struct {
		Web string `json:"web"`
	} `json:"_links"`
}

type JiraResponse struct {
	Expands    []string `json:"_expands"`
	Size       int      `json:"size"`
	Start      int      `json:"start"`
	Limit      int      `json:"limit"`
	IsLastPage bool     `json:"isLastPage"`
	Links      struct {
		Base    string `json:"base"`
		Context string `json:"context"`
	} `json:"_links"`
	Values []Issue `json:"values"`
}
