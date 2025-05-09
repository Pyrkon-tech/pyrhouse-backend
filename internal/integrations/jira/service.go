package jira

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

type JiraService struct {
	baseURL       string
	email         string
	token         string
	serviceDeskID string
}

func NewJiraService() *JiraService {
	_ = godotenv.Load()

	baseURL := os.Getenv("JIRA_BASE_URL")
	email := os.Getenv("JIRA_EMAIL")
	token := os.Getenv("JIRA_API_TOKEN")
	serviceDeskID := os.Getenv("JIRA_SERVICE_DESK_ID")

	return &JiraService{
		baseURL:       baseURL,
		email:         email,
		token:         token,
		serviceDeskID: serviceDeskID,
	}
}

func (s *JiraService) GetTasks(status string, limit string, start string) ([]Issue, error) {
	url := fmt.Sprintf("%s/rest/servicedeskapi/request?serviceDeskId=%s&limit=%s&start=%s",
		s.baseURL, s.serviceDeskID, limit, start)

	if status != "" {
		url += fmt.Sprintf("&status=%s", status)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(s.email, s.token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("jira returned %s", resp.Status)
	}

	var response JiraResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response.Values, nil
}

func (s *JiraService) GetComments(issueID string, limit string, start string) ([]Comment, error) {
	url := fmt.Sprintf("%s/rest/servicedeskapi/request/%s/comment?limit=%s&start=%s",
		s.baseURL, issueID, limit, start)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(s.email, s.token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("jira returned %s", resp.Status)
	}

	var response CommentsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response.Comments, nil
}

func (s *JiraService) GetTaskWithComments(issueID string) (*Issue, error) {
	issue, err := s.getTask(issueID)
	if err != nil {
		return nil, err
	}

	comments, err := s.GetComments(issueID, "100", "0")
	if err != nil {
		return nil, err
	}

	issue.Comments = comments
	return issue, nil
}

func (s *JiraService) getTask(issueID string) (*Issue, error) {
	url := fmt.Sprintf("%s/rest/servicedeskapi/request/%s", s.baseURL, issueID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(s.email, s.token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("jira returned %s", resp.Status)
	}

	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, err
	}

	return &issue, nil
}
