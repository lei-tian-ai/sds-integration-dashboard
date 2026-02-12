package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const buildkiteBaseURL = "https://api.buildkite.com/v2"
const buildkiteMaxPages = 10 // Fetch up to 10 pages (1000 builds)
const buildkitePerPage = 100

func buildkiteConfig() (token, org string, ok bool) {
	token = strings.TrimSpace(os.Getenv("BUILDKITE_TOKEN"))
	org = strings.TrimSpace(os.Getenv("BUILDKITE_ORG"))
	if token == "" || org == "" {
		return "", "", false
	}
	return token, org, true
}

func buildkiteConfigMissing() []string {
	var missing []string
	if strings.TrimSpace(os.Getenv("BUILDKITE_TOKEN")) == "" {
		missing = append(missing, "BUILDKITE_TOKEN")
	}
	if strings.TrimSpace(os.Getenv("BUILDKITE_ORG")) == "" {
		missing = append(missing, "BUILDKITE_ORG")
	}
	return missing
}

// BuildKite Build response structure
type BuildkiteBuild struct {
	ID          string    `json:"id"`
	Number      int       `json:"number"`
	State       string    `json:"state"` // passed, failed, canceled, running, scheduled
	StartedAt   string    `json:"started_at"`
	FinishedAt  string    `json:"finished_at"`
	CreatedAt   string    `json:"created_at"`
	ScheduledAt string    `json:"scheduled_at"`
	Pipeline    struct {
		Slug string `json:"slug"`
		Name string `json:"name"`
	} `json:"pipeline"`
	Branch  string `json:"branch"`
	Commit  string `json:"commit"`
	Message string `json:"message"`
}

// fetchBuilds fetches builds from BuildKite API with pagination
// For deployment pipeline, fetch from specific pipeline endpoint instead of org-wide
func fetchBuilds(c *gin.Context, token, org string, createdFrom time.Time) ([]BuildkiteBuild, error) {
	var allBuilds []BuildkiteBuild

	// Fetch directly from the deployment pipeline for better filtering
	pipeline := "core-stack-deployment-pipeline-legacy"

	for page := 1; page <= buildkiteMaxPages; page++ {
		query := url.Values{}
		query.Set("created_from", createdFrom.Format(time.RFC3339))
		query.Set("per_page", fmt.Sprintf("%d", buildkitePerPage))
		query.Set("page", fmt.Sprintf("%d", page))

		// Query specific pipeline instead of all org builds
		reqURL := fmt.Sprintf("%s/organizations/%s/pipelines/%s/builds?%s", buildkiteBaseURL, org, pipeline, query.Encode())
		req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("BuildKite API returned %d: %s", resp.StatusCode, string(body))
		}

		var builds []BuildkiteBuild
		if err := json.Unmarshal(body, &builds); err != nil {
			return nil, err
		}

		if len(builds) == 0 {
			break
		}

		allBuilds = append(allBuilds, builds...)

		// If we got fewer than per_page, we've reached the last page
		if len(builds) < buildkitePerPage {
			break
		}

		log.Printf("[BuildKite] Fetched page %d (%d builds, %d total)", page, len(builds), len(allBuilds))
	}

	log.Printf("[BuildKite] Total builds fetched: %d", len(allBuilds))
	return allBuilds, nil
}

// isDeploymentPipeline checks if a build is from a deployment pipeline
// Configured to track: Core Stack Deployment Pipeline Legacy
func isDeploymentPipeline(build BuildkiteBuild) bool {
	slug := strings.ToLower(build.Pipeline.Slug)

	// Track Core Stack Deployment Pipeline Legacy
	if slug == "core-stack-deployment-pipeline-legacy" {
		return true
	}

	return false
}

// kpiBuildkiteDeploymentTime returns average deployment time per week
func kpiBuildkiteDeploymentTime(c *gin.Context) {
	token, org, ok := buildkiteConfig()
	if !ok {
		missing := buildkiteConfigMissing()
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "BuildKite not configured",
			"missing": missing,
			"hint":    "Set BUILDKITE_TOKEN and BUILDKITE_ORG in .env. See docs/buildkite-setup.md",
		})
		return
	}

	// Fetch builds from last 3 months
	threeMonthsAgo := time.Now().AddDate(0, -3, 0)
	builds, err := fetchBuilds(c, token, org, threeMonthsAgo)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch builds: " + err.Error()})
		return
	}

	// Filter deployment builds and calculate durations by week
	weekDurations := make(map[string][]float64) // week -> list of durations in minutes
	deploymentCount := 0

	for _, build := range builds {
		// Only count passed deployments for average time
		if build.State != "passed" {
			continue
		}

		if !isDeploymentPipeline(build) {
			continue
		}

		startedAt, okStart := parseTime(build.StartedAt)
		finishedAt, okFinish := parseTime(build.FinishedAt)
		if !okStart || !okFinish || finishedAt.Before(startedAt) {
			continue
		}

		durationMinutes := finishedAt.Sub(startedAt).Minutes()
		week := weekKey(finishedAt)
		weekDurations[week] = append(weekDurations[week], durationMinutes)
		deploymentCount++
	}

	log.Printf("[BuildKite] Deployment time: %d deployment builds processed", deploymentCount)

	// Calculate average per week
	var weeks []string
	for w := range weekDurations {
		weeks = append(weeks, w)
	}
	sort.Strings(weeks)

	avgDurations := make([]float64, len(weeks))
	for i, w := range weeks {
		durations := weekDurations[w]
		var sum float64
		for _, d := range durations {
			sum += d
		}
		avgDurations[i] = sum / float64(len(durations))
	}

	c.JSON(http.StatusOK, gin.H{
		"weeks":             weeks,
		"avg_duration_mins": avgDurations,
		"meta": gin.H{
			"total_builds":       len(builds),
			"deployment_builds":  deploymentCount,
			"date_range":         fmt.Sprintf("last 3 months (from %s)", threeMonthsAgo.Format("2006-01-02")),
			"note":               "Average deployment time (start to finish) for passed builds only",
			"org":                org,
		},
	})
}

// kpiBuildkiteDeploymentFailureRate returns deployment failure rate per week
func kpiBuildkiteDeploymentFailureRate(c *gin.Context) {
	token, org, ok := buildkiteConfig()
	if !ok {
		missing := buildkiteConfigMissing()
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "BuildKite not configured",
			"missing": missing,
			"hint":    "Set BUILDKITE_TOKEN and BUILDKITE_ORG in .env. See docs/buildkite-setup.md",
		})
		return
	}

	// Fetch builds from last 3 months
	threeMonthsAgo := time.Now().AddDate(0, -3, 0)
	builds, err := fetchBuilds(c, token, org, threeMonthsAgo)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch builds: " + err.Error()})
		return
	}

	// Count passed and failed deployments by week
	weekPassed := make(map[string]int)
	weekFailed := make(map[string]int)
	deploymentCount := 0

	for _, build := range builds {
		if !isDeploymentPipeline(build) {
			continue
		}

		// Only count finished builds (passed or failed)
		if build.State != "passed" && build.State != "failed" {
			continue
		}

		finishedAt, ok := parseTime(build.FinishedAt)
		if !ok {
			continue
		}

		week := weekKey(finishedAt)
		if build.State == "passed" {
			weekPassed[week]++
		} else if build.State == "failed" {
			weekFailed[week]++
		}
		deploymentCount++
	}

	log.Printf("[BuildKite] Failure rate: %d deployment builds processed", deploymentCount)

	// Calculate failure rate per week
	weeksMap := make(map[string]struct{})
	for w := range weekPassed {
		weeksMap[w] = struct{}{}
	}
	for w := range weekFailed {
		weeksMap[w] = struct{}{}
	}

	var weeks []string
	for w := range weeksMap {
		weeks = append(weeks, w)
	}
	sort.Strings(weeks)

	failureRates := make([]float64, len(weeks))
	passedCounts := make([]int, len(weeks))
	failedCounts := make([]int, len(weeks))

	for i, w := range weeks {
		passed := weekPassed[w]
		failed := weekFailed[w]
		total := passed + failed

		passedCounts[i] = passed
		failedCounts[i] = failed

		if total > 0 {
			failureRates[i] = float64(failed) / float64(total) * 100
		} else {
			failureRates[i] = 0
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"weeks":         weeks,
		"failure_rate":  failureRates, // percentage
		"passed":        passedCounts,
		"failed":        failedCounts,
		"meta": gin.H{
			"total_builds":       len(builds),
			"deployment_builds":  deploymentCount,
			"date_range":         fmt.Sprintf("last 3 months (from %s)", threeMonthsAgo.Format("2006-01-02")),
			"note":               "Failure rate = failed / (passed + failed) * 100",
			"org":                org,
		},
	})
}
