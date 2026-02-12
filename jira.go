package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// JIRA REST API v3 types (subset we need for search)
// https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-issue-search/

type jiraSearchResponse struct {
	Issues []jiraIssue `json:"issues"`
	Total  int         `json:"total"`
}

type jiraIssue struct {
	Key    string     `json:"key"`
	Fields jiraFields `json:"fields"`
}

type jiraFields struct {
	Summary string     `json:"summary"`
	Status  jiraStatus `json:"status"`
	Created string     `json:"created"`
	Updated string     `json:"updated"`
}

type jiraStatus struct {
	Name string `json:"name"`
}

// JIRAIssue is the simplified shape we return to the frontend
type JIRAIssue struct {
	Key      string `json:"key"`
	Summary  string `json:"summary"`
	Status   string `json:"status"`
	Created  string `json:"created"`
	Updated  string `json:"updated"`
}

func jiraConfig() (baseURL, email, token string, ok bool) {
	domain := strings.TrimSpace(os.Getenv("JIRA_DOMAIN"))
	email = strings.TrimSpace(os.Getenv("JIRA_EMAIL"))
	token = strings.TrimSpace(os.Getenv("JIRA_API_TOKEN"))
	if domain == "" || email == "" || token == "" {
		return "", "", "", false
	}
	baseURL = "https://" + domain + ".atlassian.net"
	return baseURL, email, token, true
}

func jiraConfigMissing() []string {
	var missing []string
	if strings.TrimSpace(os.Getenv("JIRA_DOMAIN")) == "" {
		missing = append(missing, "JIRA_DOMAIN")
	}
	if strings.TrimSpace(os.Getenv("JIRA_EMAIL")) == "" {
		missing = append(missing, "JIRA_EMAIL")
	}
	if strings.TrimSpace(os.Getenv("JIRA_API_TOKEN")) == "" {
		missing = append(missing, "JIRA_API_TOKEN")
	}
	return missing
}

func jiraSearch(c *gin.Context) {
	baseURL, email, token, ok := jiraConfig()
	if !ok {
		missing := jiraConfigMissing()
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "JIRA not configured",
			"missing": missing,
			"hint":    "Export JIRA_DOMAIN, JIRA_EMAIL, and JIRA_API_TOKEN in the same terminal before running the backend",
		})
		return
	}

	// Default JQL must include a restriction (e.g. date or project); unbounded queries return 400
	jql := c.DefaultQuery("jql", "created >= -180d order by created DESC")
	maxResults := c.DefaultQuery("maxResults", "50")

	// Use /rest/api/3/search/jql (old /rest/api/3/search removed, CHANGE-2046)
	apiURL := baseURL + "/rest/api/3/search/jql?" + url.Values{
		"jql":        {jql},
		"maxResults": {maxResults},
		"fields":     {"summary,status,created,updated"},
	}.Encode()

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, apiURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	auth := base64.StdEncoding.EncodeToString([]byte(email + ":" + token))
	req.Header.Set("Authorization", "Basic "+auth)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "JIRA request failed: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, gin.H{
			"error":  fmt.Sprintf("JIRA API returned %d", resp.StatusCode),
			"detail": string(body),
		})
		return
	}

	var search jiraSearchResponse
	if err := json.Unmarshal(body, &search); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid JIRA response: " + err.Error()})
		return
	}

	issues := make([]JIRAIssue, 0, len(search.Issues))
	for _, i := range search.Issues {
		issues = append(issues, JIRAIssue{
			Key:     i.Key,
			Summary: i.Fields.Summary,
			Status:  i.Fields.Status.Name,
			Created: i.Fields.Created,
			Updated: i.Fields.Updated,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"total":  search.Total,
		"issues": issues,
	})
}
