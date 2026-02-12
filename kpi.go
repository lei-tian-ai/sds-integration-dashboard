package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Time-in-build KPI: filter 22515 (vehicle build epics).
// Rogue = days from first VBUILD ticket In Progress to last ticket Done.
// MachE = days from epic opened to "release to fleet" ticket closed.
// X-axis: calendar week; Y-axis: average days (two lines: Rogue, MachE).

const (
	kpiFilterIDDefault = "22515"
	kpiMaxEpics        = 100
	kpiMaxChildren     = 30
	kpiCreatedDays     = 730 // 2 years so we get enough closed epics for trend
)

// jiraAPI runs an authenticated request to JIRA.
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

// jiraAPIReqPost sends a POST request with JSON body (e.g. for /rest/api/3/search to avoid URL length limits).
func jiraAPIReqPost(c *gin.Context, baseURL, email, token, path string, body interface{}) (*http.Response, []byte, error) {
	rawURL := baseURL + path
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, nil, err
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, rawURL, strings.NewReader(string(jsonBody)))
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
	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp, respBody, nil
}

// getFilter returns the JQL for a saved filter.
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

// searchJQL returns issues from /rest/api/3/search/jql with requested fields and expand.
// startAt is the 0-based index for pagination (use 0 for first page).
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

// searchJQLWithTotal is like searchJQL but also returns the total count from the API response when present (for validation).
func searchJQLWithTotal(c *gin.Context, baseURL, email, token, jql string, fields []string, maxResults, startAt int, expand string) ([]map[string]interface{}, *int, error) {
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
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("search: %d %s", resp.StatusCode, string(body))
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, nil, err
	}
	var total *int
	if t, ok := raw["total"].(float64); ok {
		n := int(t)
		total = &n
	}
	var issues []map[string]interface{}
	if iss, ok := raw["issues"].([]interface{}); ok && len(iss) > 0 {
		for _, i := range iss {
			if m, ok := i.(map[string]interface{}); ok {
				issues = append(issues, m)
			}
		}
		return issues, total, nil
	}
	if vals, ok := raw["values"].([]interface{}); ok {
		for _, v := range vals {
			if m, ok := v.(map[string]interface{}); ok {
				issues = append(issues, m)
			}
		}
		return issues, total, nil
	}
	return nil, nil, fmt.Errorf("unexpected response shape")
}

const vosSearchMaxRetries = 2
const vosSearchBackoffSec = 3

// searchJQLWithTotalRateLimited calls searchJQLWithTotal and retries on 429 (rate limit) with backoff.
// On final failure it returns (nil, nil, err, attempts) so the handler can show JIRA response and retry count.
func searchJQLWithTotalRateLimited(c *gin.Context, baseURL, email, token, jql string, fields []string, maxResults, startAt int, expand string) ([]map[string]interface{}, *int, error, int) {
	var lastErr error
	attempts := 0
	for attempt := 0; attempt < vosSearchMaxRetries; attempt++ {
		attempts = attempt + 1
		if attempt > 0 {
			backoff := time.Duration(attempt*vosSearchBackoffSec) * time.Second
			log.Printf("[VOS] 429 rate limited; retrying in %v (attempt %d/%d)", backoff, attempt+1, vosSearchMaxRetries)
			time.Sleep(backoff)
		}
		page, total, err := searchJQLWithTotal(c, baseURL, email, token, jql, fields, maxResults, startAt, expand)
		if err == nil {
			return page, total, nil, 0
		}
		log.Printf("[VOS] JIRA request startAt=%d failed: %v", startAt, err)
		if !strings.Contains(err.Error(), "429") {
			return nil, nil, err, attempts
		}
		lastErr = err
	}
	return nil, nil, lastErr, attempts
}

// searchJIRAPost runs POST /rest/api/3/search/jql with JQL in the body. (POST /rest/api/3/search returns 410 removed.)
// If you get 400 Invalid request payload, use GET searchJQLWithTotal instead (VOS does).
func searchJIRAPost(c *gin.Context, baseURL, email, token, jql string, fields []string, maxResults, startAt int) ([]map[string]interface{}, *int, error) {
	body := map[string]interface{}{
		"jql":        jql,
		"maxResults": maxResults,
		"startAt":    startAt,
		"fields":     fields,
	}
	resp, respBody, err := jiraAPIReqPost(c, baseURL, email, token, "/rest/api/3/search/jql", body)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("search: %d %s", resp.StatusCode, string(respBody))
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, nil, err
	}
	var total *int
	if t, ok := raw["total"].(float64); ok {
		n := int(t)
		total = &n
	}
	var issues []map[string]interface{}
	if iss, ok := raw["issues"].([]interface{}); ok {
		for _, i := range iss {
			if m, ok := i.(map[string]interface{}); ok {
				issues = append(issues, m)
			}
		}
	}
	if len(issues) == 0 && raw["values"] != nil {
		if vals, ok := raw["values"].([]interface{}); ok {
			for _, v := range vals {
				if m, ok := v.(map[string]interface{}); ok {
					issues = append(issues, m)
				}
			}
		}
	}
	return issues, total, nil
}

// searchJIRAPostRateLimited is like searchJIRAPost but retries on 429. Used for VOS so JQL is in body (no URL truncation).
func searchJIRAPostRateLimited(c *gin.Context, baseURL, email, token, jql string, fields []string, maxResults, startAt int) ([]map[string]interface{}, *int, error, int) {
	var lastErr error
	attempts := 0
	for attempt := 0; attempt < vosSearchMaxRetries; attempt++ {
		attempts = attempt + 1
		if attempt > 0 {
			backoff := time.Duration(attempt*vosSearchBackoffSec) * time.Second
			log.Printf("[VOS] 429 rate limited; retrying in %v (attempt %d/%d)", backoff, attempt+1, vosSearchMaxRetries)
			time.Sleep(backoff)
		}
		page, total, err := searchJIRAPost(c, baseURL, email, token, jql, fields, maxResults, startAt)
		if err == nil {
			return page, total, nil, 0
		}
		log.Printf("[VOS] JIRA POST request startAt=%d failed: %v", startAt, err)
		if !strings.Contains(err.Error(), "429") {
			return nil, nil, err, attempts
		}
		lastErr = err
	}
	return nil, nil, lastErr, attempts
}

// getIssue returns a single issue with optional expand (e.g. changelog).
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
	// path like "fields.summary" or "fields.status.name"
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

// statusTransitionFromChangelog returns the time when the issue first entered statusName (e.g. "In Progress", "Done").
func statusTransitionFromChangelog(issue map[string]interface{}, statusName string) (time.Time, bool) {
	return statusTransitionFromChangelogAny(issue, []string{statusName})
}

// statusTransitionFromChangelogAny returns the time when the issue first (earliest) entered any of the given status names.
// Changelog histories are typically oldest-first; we iterate 0..len-1 to get the first transition.
func statusTransitionFromChangelogAny(issue map[string]interface{}, statusNames []string) (time.Time, bool) {
	changelog, _ := issue["changelog"].(map[string]interface{})
	if changelog == nil {
		return time.Time{}, false
	}
	histories, _ := changelog["histories"].([]interface{})
	if len(histories) == 0 {
		return time.Time{}, false
	}
	// Oldest to newest so we get the first transition to this status
	for i := 0; i < len(histories); i++ {
		h, _ := histories[i].(map[string]interface{})
		if h == nil {
			continue
		}
		created, _ := h["created"].(string)
		t, ok := parseTime(created)
		if !ok {
			continue
		}
		items, _ := h["items"].([]interface{})
		for _, it := range items {
			item, _ := it.(map[string]interface{})
			if item == nil {
				continue
			}
			if item["field"] == "status" {
				to, _ := item["toString"].(string)
				for _, name := range statusNames {
					if strings.EqualFold(to, name) {
						return t, true
					}
				}
			}
		}
	}
	return time.Time{}, false
}

// isRogueEpic returns true if the epic name contains "ROG" (Rogue build).
func isRogueEpic(epic map[string]interface{}) bool {
	summary := getFieldString(epic, "fields.summary")
	return strings.Contains(strings.ToUpper(summary), "ROG")
}

// isMachEEpic returns true if the epic is a MachE build (name contains "MCE"), but not D-Max/DMX.
func isMachEEpic(epic map[string]interface{}) bool {
	summary := getFieldString(epic, "fields.summary")
	upper := strings.ToUpper(summary)
	// Exclude D-Max / DMAX / DMX so they are not counted as MachE
	if strings.Contains(upper, "D-MAX") || strings.Contains(upper, "DMAX") || strings.Contains(upper, "DMX-") {
		return false
	}
	return strings.Contains(upper, "MCE")
}

func getLabels(issue map[string]interface{}) []string {
	fields, _ := issue["fields"].(map[string]interface{})
	if fields == nil {
		return nil
	}
	labels, _ := fields["labels"].([]interface{})
	var out []string
	for _, l := range labels {
		if s, ok := l.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// isVBUILD returns true if the issue is a VBUILD ticket: same project (key VBUILD-*) or summary suggests vehicle build.
func isVBUILD(issue map[string]interface{}) bool {
	key, _ := issue["key"].(string)
	if strings.HasPrefix(strings.ToUpper(key), "VBUILD-") {
		return true
	}
	summary := strings.ToLower(getFieldString(issue, "fields.summary"))
	return strings.Contains(summary, "vbuild") || strings.Contains(summary, "v-build") || strings.Contains(summary, "vehicle build")
}

// isReleaseToFleet returns true if the issue is the "release to fleet" ticket.
func isReleaseToFleet(issue map[string]interface{}) bool {
	summary := strings.ToLower(getFieldString(issue, "fields.summary"))
	return strings.Contains(summary, "release to fleet") ||
		(strings.Contains(summary, "release") && strings.Contains(summary, "fleet")) ||
		strings.Contains(summary, "released to fleet")
}

func weekKey(t time.Time) string {
	year, week := t.ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, week)
}

// extractVehicleName returns the vehicle/epic name from summary (e.g. "ROG-131", "MCE-203").
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

// stripOrderBy removes trailing " ORDER BY ..." from JQL so it can be wrapped in () safely.
func stripOrderBy(jql string) string {
	j := strings.TrimSpace(jql)
	upper := strings.ToUpper(j)
	idx := strings.LastIndex(upper, " ORDER BY ")
	if idx == -1 {
		return jql
	}
	return strings.TrimSpace(j[:idx])
}

// stripOpenOnly removes open-only restrictions so we get closed epics too (for trend over time).
func stripOpenOnly(jql string) string {
	s := " " + strings.ToUpper(jql) + " "
	// Resolution-based: remove so resolved epics are included
	for _, phrase := range []string{" AND RESOLUTION IS EMPTY ", " AND RESOLUTION = UNRESOLVED ", " RESOLUTION IS EMPTY AND ", " RESOLUTION = UNRESOLVED AND "} {
		s = strings.ReplaceAll(s, phrase, " ")
	}
	// Status-based: remove "status not in (Done, Closed)" / "status in (Open, In Progress)" style
	for _, phrase := range []string{" AND STATUS NOT IN (DONE, CLOSED) ", " AND STATUS IN (OPEN, \"IN PROGRESS\") ", " STATUS NOT IN (DONE, CLOSED) AND "} {
		s = strings.ReplaceAll(s, phrase, " ")
	}
	return strings.TrimSpace(s)
}

// kpiDebugEpic processes a single epic (e.g. VBUILD-5762) and returns build time and step-by-step details for validation.
func kpiDebugEpic(c *gin.Context) {
	baseURL, email, token, ok := jiraConfig()
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "JIRA not configured", "missing": jiraConfigMissing()})
		return
	}
	key := strings.TrimSpace(strings.ToUpper(c.DefaultQuery("epic", c.Query("key"))))
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing query param: epic= or key= (e.g. epic=VBUILD-5762)"})
		return
	}

	// 1. Fetch epic with changelog
	epic, err := getIssue(c, baseURL, email, token, key, "changelog")
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "fetch epic: " + err.Error(), "epic_key": key})
		return
	}
	summary := getFieldString(epic, "fields.summary")
	epicCreated, hasEpicCreated := getFieldTime(epic, "fields.created")
	isRogue := isRogueEpic(epic)
	isMachE := isMachEEpic(epic)

	// 2. Get children (parent = key or parentEpic = key)
	childJQL := "parent = " + key
	children, err := searchJQL(c, baseURL, email, token, childJQL,
		[]string{"summary", "status", "created", "updated"}, kpiMaxChildren, 0, "")
	if err != nil {
		childJQL = "parentEpic = " + key
		children, err = searchJQL(c, baseURL, email, token, childJQL,
			[]string{"summary", "status", "created", "updated"}, kpiMaxChildren, 0, "")
	}
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"epic_key":       key,
			"summary":        summary,
			"is_rogue":       isRogue,
			"is_mach_e":      isMachE,
			"epic_created":   formatTime(epicCreated),
			"children_count": 0,
			"error":          "no children: " + err.Error(),
			"build_days":     nil,
			"week":           nil,
		})
		return
	}

	// 3. For Rogue: first VBUILD child In Progress → last VBUILD child Done
	var childDetails []gin.H
	var firstInProgress, lastDone time.Time
	for _, ch := range children {
		childKey, _ := ch["key"].(string)
		chSummary := getFieldString(ch, "fields.summary")
		isVbuild := isVBUILD(ch)
		detail := gin.H{"key": childKey, "summary": chSummary, "is_vbuild": isVbuild}
		if childKey == "" {
			childDetails = append(childDetails, detail)
			continue
		}
		issue, err := getIssue(c, baseURL, email, token, childKey, "changelog")
		if err != nil {
			detail["error"] = err.Error()
			childDetails = append(childDetails, detail)
			continue
		}
		var firstIP, firstD time.Time
		if t, ok := statusTransitionFromChangelogAny(issue, []string{"In Progress", "In progress"}); ok {
			firstIP = t
			if isVbuild && (firstInProgress.IsZero() || t.Before(firstInProgress)) {
				firstInProgress = t
			}
		}
		if t, ok := statusTransitionFromChangelogAny(issue, []string{"Done", "Closed", "Complete", "Resolved"}); ok {
			firstD = t
			if isVbuild && t.After(lastDone) {
				lastDone = t
			}
		}
		detail["first_in_progress"] = formatTime(firstIP)
		detail["first_done"] = formatTime(firstD)
		childDetails = append(childDetails, detail)
	}

	var buildDays interface{}
	var week interface{}
	if !firstInProgress.IsZero() && !lastDone.IsZero() && lastDone.After(firstInProgress) {
		d := lastDone.Sub(firstInProgress).Hours() / 24
		buildDays = float64(int(d*10)) / 10
		week = weekKey(lastDone)
	} else if hasEpicCreated && !isRogue && !isMachE {
		// All metric: epic created → resolved
		if epicDone, ok := statusTransitionFromChangelogAny(epic, []string{"Done", "Closed", "Complete", "Resolved"}); ok && epicDone.After(epicCreated) {
			buildDays = epicDone.Sub(epicCreated).Hours() / 24
			week = weekKey(epicDone)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"epic_key":          key,
		"summary":           summary,
		"is_rogue":          isRogue,
		"is_mach_e":         isMachE,
		"epic_created":      formatTime(epicCreated),
		"children_count":   len(children),
		"children":          childDetails,
		"first_in_progress": formatTime(firstInProgress),
		"last_done":         formatTime(lastDone),
		"build_days":        buildDays,
		"week":              week,
	})
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02T15:04:05Z07:00")
}

// kpiTimeInBuild returns time series: by week, average days for Rogue and MachE.
func kpiTimeInBuild(c *gin.Context) {
	baseURL, email, token, ok := jiraConfig()
	if !ok {
		missing := jiraConfigMissing()
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "JIRA not configured",
			"missing": missing,
		})
		return
	}

	var epicJQL string
	var filterID string
	if customJQL := strings.TrimSpace(c.Query("jql")); customJQL != "" {
		// Use provided JQL (e.g. project in (10525) AND 'issue' in portfolioChildIssuesOf(VBUILD-8121)); ensure we get epics only
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
		// Fetch epics from filter (include closed so we get trend over time).
		// Strip "resolution is empty" so we get both open and closed epics; strip ORDER BY for safe wrapping.
		epicJQL = stripOpenOnly(stripOrderBy(jql))
		epicJQL = "(" + epicJQL + ") AND issuetype = Epic"
		if !strings.Contains(strings.ToLower(epicJQL), "created") {
			epicJQL = "(" + epicJQL + ") AND created >= -" + fmt.Sprintf("%dd", kpiCreatedDays)
		}
		// Optional: include project(s) in addition to filter, e.g. project_keys=VBUILD so VBUILD epics are included
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
	// Paginate to fetch all matching epics (so we get closed ones across many weeks)
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

	// Optional: include specific epic keys (e.g. VBUILD-4243) so they appear in table/chart even if not in JQL
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
		week       string
		days       float64
		epicKey    string
		summary    string
		startTime  time.Time
		finishTime time.Time
	}
	type machEPoint struct {
		week       string
		days       float64
		epicKey    string
		summary    string
		startTime  time.Time
		finishTime time.Time
	}
	type allPoint struct {
		week       string
		days       float64
		epicKey    string
		summary    string
		startTime  time.Time
		finishTime time.Time
	}
	var roguePoints []roguePoint
	var machEPoints []machEPoint
	var allPoints []allPoint

	// Approximation: use only epic-level data (created → resolutiondate). No child tickets or changelogs — much faster.
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
			roguePoints = append(roguePoints, roguePoint{week, days, key, epicSummary, epicCreated, epicResolved})
		} else if isMachEEpic(epic) {
			machEPoints = append(machEPoints, machEPoint{week, days, key, epicSummary, epicCreated, epicResolved})
		} else {
			allPoints = append(allPoints, allPoint{week, days, key, epicSummary, epicCreated, epicResolved})
		}
	}

	// Build epic_rows for the table: every finished epic with start/finish/build_days, sorted by finish time
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

	// Per-week, per-type vehicle names so labels go next to the right series (Rogue/MachE/Other)
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

	// Aggregate by week: average days per week
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

// JQL for tickets assigned to Vehicle OS engineers during build (VOS integration team). Matches JIRA filter exactly.
const vosTicketsJQL = `project in (10525) AND 'issue' in portfolioChildIssuesOf(VBUILD-8121) and assignee in membersOf("okta-team-vos_si")`

// JQL for KPI #4: Build Issues Caught After Release to Calibration (bugs in VBUILD portfolio)
const buildBugsJQL = `project in (10525) AND 'issue' in portfolioChildIssuesOf(VBUILD-8121) AND type in ("Bug", "Bug Report")`

// JQL for MTBF (Mean Time Between Failure): Vehicle Stability Issue Reports
const mtbfJQL = `project = VSTAB AND type = "Vehicle Stability Issue Report" AND component = "On Road Dev"`

const vosTicketsMaxResults = 100  // JIRA caps per-page at 100
const vosTicketsCreatedDays = 365 // we keep only issues created in last 365 days (~430)
const vosTicketsPageDelay = 400 * time.Millisecond
const vosTicketsInRangeCap = 2000  // stop when we have this many in-range issues (safety cap)
const vosTicketsMaxPages = 25      // max pages to fetch (2500 raw) with date filter in JQL

// kpiVOSTickets returns tickets assigned to Vehicle OS engineers during build: by week, tickets created and tickets resolved.
// Uses week-by-week queries to avoid JIRA API pagination bugs and improve performance.
func kpiVOSTickets(c *gin.Context) {
	baseURL, email, token, ok := jiraConfig()
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "JIRA not configured", "missing": jiraConfigMissing()})
		return
	}

	baseJQL := vosTicketsJQL
	log.Printf("[VOS] Base JQL: %s", baseJQL)
	log.Printf("[VOS] Fetching issues week-by-week for last 2 months")

	// Generate week ranges for the last 2 months
	now := time.Now()
	twoMonthsAgo := now.AddDate(0, -2, 0)

	// Find the start of the week 2 months ago (Monday)
	startDate := twoMonthsAgo
	for startDate.Weekday() != time.Monday {
		startDate = startDate.AddDate(0, 0, -1)
	}

	// Collect all week ranges first
	type weekRange struct {
		start   time.Time
		end     time.Time
		weekKey string
	}
	var weekRanges []weekRange
	for weekStart := startDate; weekStart.Before(now); weekStart = weekStart.AddDate(0, 0, 7) {
		weekEnd := weekStart.AddDate(0, 0, 7)
		weekRanges = append(weekRanges, weekRange{
			start:   weekStart,
			end:     weekEnd,
			weekKey: weekKey(weekStart),
		})
	}

	log.Printf("[VOS] Querying %d weeks in parallel...", len(weekRanges))

	// Run queries in parallel using goroutines
	type result struct {
		weekKey  string
		created  int
		resolved int
		err      error
	}

	results := make(chan result, len(weekRanges))
	var wg sync.WaitGroup

	for _, w := range weekRanges {
		wg.Add(1)
		go func(week weekRange) {
			defer wg.Done()

			r := result{weekKey: week.weekKey}

			// Query for issues created in this week
			createdJQL := fmt.Sprintf("(%s) AND created >= '%s' AND created < '%s'",
				baseJQL,
				week.start.Format("2006-01-02"),
				week.end.Format("2006-01-02"))

			createdIssues, err := searchJQL(c, baseURL, email, token, createdJQL, []string{"key"}, 100, 0, "")
			if err != nil {
				log.Printf("[VOS] Failed to query created for week %s: %v", week.weekKey, err)
			} else {
				r.created = len(createdIssues)
			}

			// Query for issues resolved in this week
			resolvedJQL := fmt.Sprintf("(%s) AND resolutiondate >= '%s' AND resolutiondate < '%s'",
				baseJQL,
				week.start.Format("2006-01-02"),
				week.end.Format("2006-01-02"))

			resolvedIssues, err := searchJQL(c, baseURL, email, token, resolvedJQL, []string{"key"}, 100, 0, "")
			if err != nil {
				log.Printf("[VOS] Failed to query resolved for week %s: %v", week.weekKey, err)
			} else {
				r.resolved = len(resolvedIssues)
			}

			results <- r
		}(w)
	}

	// Close results channel after all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	weekCreated := make(map[string]int)
	weekResolved := make(map[string]int)
	totalIssuesSeen := 0

	for r := range results {
		weekCreated[r.weekKey] = r.created
		weekResolved[r.weekKey] = r.resolved
		totalIssuesSeen += r.created
	}

	log.Printf("[VOS] Fetched data for %d weeks (total issues seen: %d)", len(weekCreated), totalIssuesSeen)

	// Build sorted list of weeks
	weeksMap := make(map[string]struct{})
	for w := range weekCreated {
		weeksMap[w] = struct{}{}
	}
	for w := range weekResolved {
		weeksMap[w] = struct{}{}
	}
	var weeks []string
	for w := range weeksMap {
		weeks = append(weeks, w)
	}
	sort.Strings(weeks)

	// Build counts arrays
	createdCounts := make([]int, len(weeks))
	resolvedCounts := make([]int, len(weeks))
	for i, w := range weeks {
		createdCounts[i] = weekCreated[w]
		resolvedCounts[i] = weekResolved[w]
	}

	meta := gin.H{
		"jql_used":    baseJQL,
		"issues_seen": totalIssuesSeen,
		"date_filter": "last 2 months (applied in JQL per-week queries)",
		"note":        fmt.Sprintf("Fetched data using week-by-week queries (much faster than fetching all %d issues)", totalIssuesSeen),
	}
	c.JSON(http.StatusOK, gin.H{
		"weeks":    weeks,
		"created":  createdCounts,
		"resolved": resolvedCounts,
		"meta":     meta,
	})
}

// kpiBuildBugs returns KPI #4: Build Issues Caught After Release to Calibration.
// Shows bugs found in VBUILD portfolio, tracked week-by-week.
func kpiBuildBugs(c *gin.Context) {
	baseURL, email, token, ok := jiraConfig()
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "JIRA not configured", "missing": jiraConfigMissing()})
		return
	}

	baseJQL := buildBugsJQL
	log.Printf("[BuildBugs] Base JQL: %s", baseJQL)
	log.Printf("[BuildBugs] Fetching bugs week-by-week for last 2 months")

	// Generate week ranges for the last 2 months
	now := time.Now()
	twoMonthsAgo := now.AddDate(0, -2, 0)

	// Find the start of the week 2 months ago (Monday)
	startDate := twoMonthsAgo
	for startDate.Weekday() != time.Monday {
		startDate = startDate.AddDate(0, 0, -1)
	}

	// Collect all week ranges first
	type weekRange struct {
		start   time.Time
		end     time.Time
		weekKey string
	}
	var weekRanges []weekRange
	for weekStart := startDate; weekStart.Before(now); weekStart = weekStart.AddDate(0, 0, 7) {
		weekEnd := weekStart.AddDate(0, 0, 7)
		weekRanges = append(weekRanges, weekRange{
			start:   weekStart,
			end:     weekEnd,
			weekKey: weekKey(weekStart),
		})
	}

	log.Printf("[BuildBugs] Querying %d weeks in parallel...", len(weekRanges))

	// Run queries in parallel using goroutines
	type result struct {
		weekKey  string
		created  int
		resolved int
		err      error
	}

	results := make(chan result, len(weekRanges))
	var wg sync.WaitGroup

	for _, w := range weekRanges {
		wg.Add(1)
		go func(week weekRange) {
			defer wg.Done()

			r := result{weekKey: week.weekKey}

			// Query for bugs created in this week
			createdJQL := fmt.Sprintf("(%s) AND created >= '%s' AND created < '%s'",
				baseJQL,
				week.start.Format("2006-01-02"),
				week.end.Format("2006-01-02"))

			createdIssues, err := searchJQL(c, baseURL, email, token, createdJQL, []string{"key"}, 100, 0, "")
			if err != nil {
				log.Printf("[BuildBugs] Failed to query created for week %s: %v", week.weekKey, err)
			} else {
				r.created = len(createdIssues)
			}

			// Query for bugs resolved in this week
			resolvedJQL := fmt.Sprintf("(%s) AND resolutiondate >= '%s' AND resolutiondate < '%s'",
				baseJQL,
				week.start.Format("2006-01-02"),
				week.end.Format("2006-01-02"))

			resolvedIssues, err := searchJQL(c, baseURL, email, token, resolvedJQL, []string{"key"}, 100, 0, "")
			if err != nil {
				log.Printf("[BuildBugs] Failed to query resolved for week %s: %v", week.weekKey, err)
			} else {
				r.resolved = len(resolvedIssues)
			}

			results <- r
		}(w)
	}

	// Close results channel after all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	weekCreated := make(map[string]int)
	weekResolved := make(map[string]int)
	totalIssuesSeen := 0

	for r := range results {
		weekCreated[r.weekKey] = r.created
		weekResolved[r.weekKey] = r.resolved
		totalIssuesSeen += r.created
	}

	log.Printf("[BuildBugs] Fetched data for %d weeks (total bugs seen: %d)", len(weekCreated), totalIssuesSeen)

	// Build sorted list of weeks
	weeksMap := make(map[string]struct{})
	for w := range weekCreated {
		weeksMap[w] = struct{}{}
	}
	for w := range weekResolved {
		weeksMap[w] = struct{}{}
	}
	var weeks []string
	for w := range weeksMap {
		weeks = append(weeks, w)
	}
	sort.Strings(weeks)

	// Build counts arrays
	createdCounts := make([]int, len(weeks))
	resolvedCounts := make([]int, len(weeks))
	for i, w := range weeks {
		createdCounts[i] = weekCreated[w]
		resolvedCounts[i] = weekResolved[w]
	}

	meta := gin.H{
		"jql_used":    baseJQL,
		"bugs_seen":   totalIssuesSeen,
		"date_filter": "last 2 months (applied in JQL per-week queries)",
		"note":        fmt.Sprintf("Fetched bug data using parallel week-by-week queries (%d bugs found)", totalIssuesSeen),
	}
	c.JSON(http.StatusOK, gin.H{
		"weeks":    weeks,
		"created":  createdCounts,
		"resolved": resolvedCounts,
		"meta":     meta,
	})
}

// kpiMTBF returns Mean Time Between Failure metric: vehicle stability issue reports.
// Tracks failure counts per week for the last 3 months.
// TODO: Add drive hours denominator once Fleetio/Neuron data source is available.
func kpiMTBF(c *gin.Context) {
	baseURL, email, token, ok := jiraConfig()
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "JIRA not configured", "missing": jiraConfigMissing()})
		return
	}

	baseJQL := mtbfJQL
	log.Printf("[MTBF] Base JQL: %s", baseJQL)
	log.Printf("[MTBF] Fetching failure reports week-by-week for last 3 months")

	// Generate week ranges for the last 3 months
	now := time.Now()
	threeMonthsAgo := now.AddDate(0, -3, 0)

	// Find the start of the week 3 months ago (Monday)
	startDate := threeMonthsAgo
	for startDate.Weekday() != time.Monday {
		startDate = startDate.AddDate(0, 0, -1)
	}

	// Collect all week ranges first
	type weekRange struct {
		start   time.Time
		end     time.Time
		weekKey string
	}
	var weekRanges []weekRange
	for weekStart := startDate; weekStart.Before(now); weekStart = weekStart.AddDate(0, 0, 7) {
		weekEnd := weekStart.AddDate(0, 0, 7)
		weekRanges = append(weekRanges, weekRange{
			start:   weekStart,
			end:     weekEnd,
			weekKey: weekKey(weekStart),
		})
	}

	log.Printf("[MTBF] Querying %d weeks in parallel...", len(weekRanges))

	// Run queries in parallel using goroutines
	type result struct {
		weekKey  string
		failures int
		err      error
	}

	results := make(chan result, len(weekRanges))
	var wg sync.WaitGroup

	for _, w := range weekRanges {
		wg.Add(1)
		go func(week weekRange) {
			defer wg.Done()

			r := result{weekKey: week.weekKey}

			// Query for failures created in this week
			createdJQL := fmt.Sprintf("(%s) AND created >= '%s' AND created < '%s'",
				baseJQL,
				week.start.Format("2006-01-02"),
				week.end.Format("2006-01-02"))

			createdIssues, err := searchJQL(c, baseURL, email, token, createdJQL, []string{"key"}, 100, 0, "")
			if err != nil {
				log.Printf("[MTBF] Failed to query failures for week %s: %v", week.weekKey, err)
			} else {
				r.failures = len(createdIssues)
			}

			results <- r
		}(w)
	}

	// Close results channel after all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	weekFailures := make(map[string]int)
	totalFailuresSeen := 0

	for r := range results {
		weekFailures[r.weekKey] = r.failures
		totalFailuresSeen += r.failures
	}

	log.Printf("[MTBF] Fetched data for %d weeks (total failures: %d)", len(weekFailures), totalFailuresSeen)

	// Build sorted list of weeks
	weeksMap := make(map[string]struct{})
	for w := range weekFailures {
		weeksMap[w] = struct{}{}
	}
	var weeks []string
	for w := range weeksMap {
		weeks = append(weeks, w)
	}
	sort.Strings(weeks)

	// Build failure counts array
	failureCounts := make([]int, len(weeks))
	for i, w := range weeks {
		failureCounts[i] = weekFailures[w]
	}

	meta := gin.H{
		"jql_used":       baseJQL,
		"failures_seen":  totalFailuresSeen,
		"date_filter":    "last 3 months (applied in JQL per-week queries)",
		"note":           "Tracking failure counts. Drive hours data source pending (Fleetio or Neuron).",
		"drive_hours":    "TODO: Add drive hours denominator",
		"data_available": "failures only",
	}
	c.JSON(http.StatusOK, gin.H{
		"weeks":    weeks,
		"failures": failureCounts,
		"meta":     meta,
	})
}

// kpiDataCollectionEfficiency returns placeholder data for Data Collection Efficiency KPI.
// TODO: Integrate with lakehouse via KunaalC's query service for real data.
// Formula: (hours of valid/usable data) / (total driving hours) * 100
// Target: >95%
func kpiDataCollectionEfficiency(c *gin.Context) {
	log.Println("[DataCollectionEfficiency] Returning placeholder data - TODO: integrate with lakehouse")

	// Generate placeholder data for the last 10 weeks
	now := time.Now()
	tenWeeksAgo := now.AddDate(0, 0, -70) // ~10 weeks

	// Find the start of the week 10 weeks ago (Monday)
	startDate := tenWeeksAgo
	for startDate.Weekday() != time.Monday {
		startDate = startDate.AddDate(0, 0, -1)
	}

	var weeks []string
	var efficiencyPercentages []float64

	// Generate mock data with some variation around 95% target
	for weekStart := startDate; weekStart.Before(now); weekStart = weekStart.AddDate(0, 0, 7) {
		weeks = append(weeks, weekKey(weekStart))
		// Mock data: efficiency between 92% and 98%
		// Add some realistic variation
		baseEfficiency := 95.0
		variation := float64((len(weeks) % 5) - 2) // -2 to +2
		efficiency := baseEfficiency + variation
		efficiencyPercentages = append(efficiencyPercentages, efficiency)
	}

	meta := gin.H{
		"data_source": "PLACEHOLDER - awaiting lakehouse integration",
		"formula":     "(valid data hours) / (total driving hours) * 100",
		"target":      ">95%",
		"status":      "TODO: Integrate with KunaalC's query service for neuron/frontier/mosaic clusters",
		"note":        "Currently returning mock data. Real implementation requires ADP auth and lakehouse query API.",
	}

	c.JSON(http.StatusOK, gin.H{
		"weeks":                 weeks,
		"efficiency_percentage": efficiencyPercentages,
		"meta":                  meta,
	})
}
