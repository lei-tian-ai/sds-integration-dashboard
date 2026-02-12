// Package main: Time-in-Build KPI handler and dependencies.
// Copy this file into your Go service and register: api.GET("/kpi/time-in-build", kpiTimeInBuild)
// Requires: JIRA_DOMAIN, JIRA_EMAIL, JIRA_API_TOKEN in env.

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// --- Constants ---
const (
	kpiFilterIDDefault = "22515"
	kpiMaxEpics        = 100
	kpiCreatedDays     = 730
)

// --- JIRA config (from jira.go) ---
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

func jiraAPIReq(c *gin.Context, baseURL, email, token, method, path string, query url.Values) (*http.Response, []byte, error) {
	rawURL := baseURL + path
	if len(query) > 0 {
		rawURL += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), method, rawURL, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(email+":"+token)))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return resp, nil, err
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp, body, nil
}

// --- Filter & search ---
func getFilter(c *gin.Context, baseURL, email, token, filterID string) (jql string, err error) {
	resp, body, err := jiraAPIReq(c, baseURL, email, token, http.MethodGet, "/rest/api/3/filter/"+filterID, nil)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("filter %s: %d %s", filterID, resp.StatusCode, string(body))
	}
	var f struct {
		JQL string `json:"jql"`
	}
	if err := json.Unmarshal(body, &f); err != nil {
		return "", err
	}
	return f.JQL, nil
}

func searchJQL(c *gin.Context, baseURL, email, token, jql string, fields []string, maxResults, startAt int, expand string) ([]map[string]interface{}, error) {
	q := url.Values{}
	q.Set("jql", jql)
	q.Set("maxResults", fmt.Sprintf("%d", maxResults))
	if startAt > 0 {
		q.Set("startAt", fmt.Sprintf("%d", startAt))
	}
	if len(fields) > 0 {
		q.Set("fields", strings.Join(fields, ","))
	}
	if expand != "" {
		q.Set("expand", expand)
	}
	resp, body, err := jiraAPIReq(c, baseURL, email, token, http.MethodGet, "/rest/api/3/search/jql", q)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search: %d %s", resp.StatusCode, string(body))
	}
	var withIssues struct {
		Issues []map[string]interface{} `json:"issues"`
	}
	if err := json.Unmarshal(body, &withIssues); err == nil && len(withIssues.Issues) > 0 {
		return withIssues.Issues, nil
	}
	var withValues struct {
		Values []map[string]interface{} `json:"values"`
	}
	if err := json.Unmarshal(body, &withValues); err != nil {
		return nil, err
	}
	return withValues.Values, nil
}

func getIssue(c *gin.Context, baseURL, email, token, key, expand string) (map[string]interface{}, error) {
	q := url.Values{}
	if expand != "" {
		q.Set("expand", expand)
	}
	resp, body, err := jiraAPIReq(c, baseURL, email, token, http.MethodGet, "/rest/api/3/issue/"+key, q)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("issue %s: %d %s", key, resp.StatusCode, string(body))
	}
	var issue map[string]interface{}
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, err
	}
	return issue, nil
}

// --- Field / time helpers ---
func parseTime(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05.000-0700", s)
	}
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func getFieldString(m map[string]interface{}, path string) string {
	parts := strings.Split(path, ".")
	cur := m
	for i, p := range parts {
		if i == len(parts)-1 {
			if v, ok := cur[p].(string); ok {
				return v
			}
			return ""
		}
		if next, ok := cur[p].(map[string]interface{}); ok {
			cur = next
		} else {
			return ""
		}
	}
	return ""
}

func getFieldTime(m map[string]interface{}, path string) (time.Time, bool) {
	s := getFieldString(m, path)
	return parseTime(s)
}

func weekKey(t time.Time) string {
	year, week := t.ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, week)
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02T15:04:05Z07:00")
}

func stripOrderBy(jql string) string {
	j := strings.TrimSpace(jql)
	upper := strings.ToUpper(j)
	idx := strings.LastIndex(upper, " ORDER BY ")
	if idx == -1 {
		return jql
	}
	return strings.TrimSpace(j[:idx])
}

func stripOpenOnly(jql string) string {
	s := " " + strings.ToUpper(jql) + " "
	for _, phrase := range []string{" AND RESOLUTION IS EMPTY ", " AND RESOLUTION = UNRESOLVED ", " RESOLUTION IS EMPTY AND ", " RESOLUTION = UNRESOLVED AND "} {
		s = strings.ReplaceAll(s, phrase, " ")
	}
	for _, phrase := range []string{" AND STATUS NOT IN (DONE, CLOSED) ", " AND STATUS IN (OPEN, \"IN PROGRESS\") ", " STATUS NOT IN (DONE, CLOSED) AND "} {
		s = strings.ReplaceAll(s, phrase, " ")
	}
	return strings.TrimSpace(s)
}

func isRogueEpic(epic map[string]interface{}) bool {
	summary := getFieldString(epic, "fields.summary")
	return strings.Contains(strings.ToUpper(summary), "ROG")
}

func isMachEEpic(epic map[string]interface{}) bool {
	summary := getFieldString(epic, "fields.summary")
	upper := strings.ToUpper(summary)
	if strings.Contains(upper, "D-MAX") || strings.Contains(upper, "DMAX") || strings.Contains(upper, "DMX-") {
		return false
	}
	return strings.Contains(upper, "MCE")
}

func extractVehicleName(summary string) string {
	s := strings.TrimSpace(summary)
	if idx := strings.Index(s, " - "); idx >= 0 {
		s = s[:idx]
	}
	parts := strings.Fields(s)
	if len(parts) > 0 {
		return strings.TrimSpace(parts[0])
	}
	return s
}

// --- Handler ---
func kpiTimeInBuild(c *gin.Context) {
	baseURL, email, token, ok := jiraConfig()
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "JIRA not configured",
			"missing": jiraConfigMissing(),
		})
		return
	}

	var epicJQL string
	var filterID string
	if customJQL := strings.TrimSpace(c.Query("jql")); customJQL != "" {
		filterID = "jql"
		epicJQL = stripOpenOnly(stripOrderBy(customJQL))
		epicJQL = "(" + epicJQL + ") AND issuetype = Epic"
		if !strings.Contains(strings.ToLower(epicJQL), "created") {
			epicJQL = "(" + epicJQL + ") AND created >= -" + fmt.Sprintf("%dd", kpiCreatedDays)
		}
	} else {
		filterID = c.DefaultQuery("filter_id", kpiFilterIDDefault)
		jql, err := getFilter(c, baseURL, email, token, filterID)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to get filter: " + err.Error()})
			return
		}
		epicJQL = stripOpenOnly(stripOrderBy(jql))
		epicJQL = "(" + epicJQL + ") AND issuetype = Epic"
		if !strings.Contains(strings.ToLower(epicJQL), "created") {
			epicJQL = "(" + epicJQL + ") AND created >= -" + fmt.Sprintf("%dd", kpiCreatedDays)
		}
		if projects := c.Query("project_keys"); projects != "" {
			var keys []string
			for _, p := range strings.Split(projects, ",") {
				p = strings.TrimSpace(strings.ToUpper(p))
				if p != "" {
					keys = append(keys, p)
				}
			}
			if len(keys) > 0 {
				extra := "issuetype = Epic AND project in (" + strings.Join(keys, ", ") + ") AND created >= -" + fmt.Sprintf("%dd", kpiCreatedDays)
				epicJQL = "(" + epicJQL + ") OR (" + extra + ")"
			}
		}
	}

	var epics []map[string]interface{}
	for startAt := 0; ; startAt += kpiMaxEpics {
		page, err := searchJQL(c, baseURL, email, token, epicJQL,
			[]string{"summary", "status", "created", "updated", "labels", "resolutiondate"}, kpiMaxEpics, startAt, "")
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "epic search: " + err.Error()})
			return
		}
		epics = append(epics, page...)
		if len(page) < kpiMaxEpics {
			break
		}
		if len(epics) >= 300 {
			break
		}
	}

	epicKeySet := make(map[string]struct{})
	for _, ep := range epics {
		if k, _ := ep["key"].(string); k != "" {
			epicKeySet[k] = struct{}{}
		}
	}
	for _, raw := range strings.Split(c.Query("include_epic_keys"), ",") {
		key := strings.TrimSpace(strings.ToUpper(raw))
		if key == "" {
			continue
		}
		if _, have := epicKeySet[key]; have {
			continue
		}
		issue, err := getIssue(c, baseURL, email, token, key, "")
		if err != nil {
			continue
		}
		epicKeySet[key] = struct{}{}
		epics = append(epics, issue)
	}

	type roguePoint struct {
		week, epicKey, summary    string
		days                      float64
		startTime, finishTime     time.Time
	}
	type machEPoint struct {
		week, epicKey, summary    string
		days                      float64
		startTime, finishTime     time.Time
	}
	type allPoint struct {
		week, epicKey, summary    string
		days                      float64
		startTime, finishTime     time.Time
	}
	var roguePoints []roguePoint
	var machEPoints []machEPoint
	var allPoints []allPoint

	for _, epic := range epics {
		key, _ := epic["key"].(string)
		if key == "" {
			continue
		}
		epicCreated, hasCreated := getFieldTime(epic, "fields.created")
		epicResolved, hasResolved := getFieldTime(epic, "fields.resolutiondate")
		if !hasCreated || !hasResolved || !epicResolved.After(epicCreated) {
			continue
		}
		days := epicResolved.Sub(epicCreated).Hours() / 24
		week := weekKey(epicResolved)
		epicSummary := getFieldString(epic, "fields.summary")

		if isRogueEpic(epic) {
			roguePoints = append(roguePoints, roguePoint{week, key, epicSummary, days, epicCreated, epicResolved})
		} else if isMachEEpic(epic) {
			machEPoints = append(machEPoints, machEPoint{week, key, epicSummary, days, epicCreated, epicResolved})
		} else {
			allPoints = append(allPoints, allPoint{week, key, epicSummary, days, epicCreated, epicResolved})
		}
	}

	type epicRow struct {
		EpicKey     string  `json:"epic_key"`
		Summary     string  `json:"summary"`
		VehicleName string  `json:"vehicle_name"`
		StartTime   string  `json:"start_time"`
		FinishTime  string  `json:"finish_time"`
		BuildDays   float64 `json:"build_days"`
		Week        string  `json:"week"`
		Type        string  `json:"type"`
	}
	var epicRows []epicRow
	for _, p := range roguePoints {
		epicRows = append(epicRows, epicRow{p.epicKey, p.summary, extractVehicleName(p.summary), formatTime(p.startTime), formatTime(p.finishTime), math.Round(p.days*10) / 10, p.week, "Rogue"})
	}
	for _, p := range machEPoints {
		epicRows = append(epicRows, epicRow{p.epicKey, p.summary, extractVehicleName(p.summary), formatTime(p.startTime), formatTime(p.finishTime), math.Round(p.days*10) / 10, p.week, "MachE"})
	}
	for _, p := range allPoints {
		epicRows = append(epicRows, epicRow{p.epicKey, p.summary, extractVehicleName(p.summary), formatTime(p.startTime), formatTime(p.finishTime), math.Round(p.days*10) / 10, p.week, "Other"})
	}
	sort.Slice(epicRows, func(i, j int) bool {
		return epicRows[i].FinishTime < epicRows[j].FinishTime
	})

	weekLabelsRogue := make(map[string][]string)
	weekLabelsMachE := make(map[string][]string)
	weekLabelsOther := make(map[string][]string)
	for _, row := range epicRows {
		if row.VehicleName == "" {
			continue
		}
		switch row.Type {
		case "Rogue":
			weekLabelsRogue[row.Week] = append(weekLabelsRogue[row.Week], row.VehicleName)
		case "MachE":
			weekLabelsMachE[row.Week] = append(weekLabelsMachE[row.Week], row.VehicleName)
		case "Other":
			weekLabelsOther[row.Week] = append(weekLabelsOther[row.Week], row.VehicleName)
		}
	}
	for _, m := range []map[string][]string{weekLabelsRogue, weekLabelsMachE, weekLabelsOther} {
		for w := range m {
			seen := make(map[string]struct{})
			var list []string
			for _, v := range m[w] {
				if _, ok := seen[v]; !ok {
					seen[v] = struct{}{}
					list = append(list, v)
				}
			}
			sort.Strings(list)
			m[w] = list
		}
	}

	rogueByWeek := make(map[string][]float64)
	for _, p := range roguePoints {
		rogueByWeek[p.week] = append(rogueByWeek[p.week], p.days)
	}
	machEByWeek := make(map[string][]float64)
	for _, p := range machEPoints {
		machEByWeek[p.week] = append(machEByWeek[p.week], p.days)
	}
	allByWeek := make(map[string][]float64)
	for _, p := range allPoints {
		allByWeek[p.week] = append(allByWeek[p.week], p.days)
	}

	weeksMap := make(map[string]struct{})
	for w := range rogueByWeek {
		weeksMap[w] = struct{}{}
	}
	for w := range machEByWeek {
		weeksMap[w] = struct{}{}
	}
	for w := range allByWeek {
		weeksMap[w] = struct{}{}
	}
	var weeks []string
	for w := range weeksMap {
		weeks = append(weeks, w)
	}
	sort.Strings(weeks)

	rogueAvg := make([]float64, len(weeks))
	machEAvg := make([]float64, len(weeks))
	allAvg := make([]float64, len(weeks))
	for i, w := range weeks {
		if vals := rogueByWeek[w]; len(vals) > 0 {
			var sum float64
			for _, v := range vals {
				sum += v
			}
			rogueAvg[i] = sum / float64(len(vals))
		}
		if vals := machEByWeek[w]; len(vals) > 0 {
			var sum float64
			for _, v := range vals {
				sum += v
			}
			machEAvg[i] = sum / float64(len(vals))
		}
		if vals := allByWeek[w]; len(vals) > 0 {
			var sum float64
			for _, v := range vals {
				sum += v
			}
			allAvg[i] = sum / float64(len(vals))
		}
	}

	epicKeys := make([]string, 0, len(epics))
	for _, ep := range epics {
		if k, _ := ep["key"].(string); k != "" {
			epicKeys = append(epicKeys, k)
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"weeks":              weeks,
		"rogue":              rogueAvg,
		"machE":              machEAvg,
		"other":              allAvg,
		"epic_rows":          epicRows,
		"week_labels_rogue":  weekLabelsRogue,
		"week_labels_mach_e": weekLabelsMachE,
		"week_labels_other":  weekLabelsOther,
		"meta": gin.H{
			"filter_id":  filterID,
			"jql_used":   epicJQL,
			"epic_keys":  epicKeys,
			"epics_seen": len(epics),
			"rogue_n":    len(roguePoints),
			"machE_n":    len(machEPoints),
			"other_n":    len(allPoints),
		},
	})
}
