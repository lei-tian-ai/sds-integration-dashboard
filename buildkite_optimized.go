package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Rate limiter for BuildKite API (200 req/min = ~3 req/sec)
var buildkiteRateLimiter = time.NewTicker(350 * time.Millisecond) // ~2.85 req/sec to be safe

// Cache for BuildKite data
var (
	buildkiteCache      *BuildKiteCacheData
	buildkiteCacheMutex sync.RWMutex
	buildkiteCacheTTL   = 5 * time.Minute
)

type BuildKiteCacheData struct {
	Builds    []BuildkiteBuild
	FetchedAt time.Time
}

func getCachedBuilds(c *gin.Context, token, org string, createdFrom time.Time) ([]BuildkiteBuild, error) {
	buildkiteCacheMutex.RLock()
	if buildkiteCache != nil && time.Since(buildkiteCache.FetchedAt) < buildkiteCacheTTL {
		builds := buildkiteCache.Builds
		buildkiteCacheMutex.RUnlock()
		log.Printf("[BuildKite Cache] Using cached data (%d builds, age: %v)", len(builds), time.Since(buildkiteCache.FetchedAt))
		return builds, nil
	}
	buildkiteCacheMutex.RUnlock()

	// Cache miss or expired, fetch new data
	builds, err := fetchBuildsParallel(c, token, org, createdFrom)
	if err != nil {
		return nil, err
	}

	// Update cache
	buildkiteCacheMutex.Lock()
	buildkiteCache = &BuildKiteCacheData{
		Builds:    builds,
		FetchedAt: time.Now(),
	}
	buildkiteCacheMutex.Unlock()
	log.Printf("[BuildKite Cache] Updated cache with %d builds", len(builds))

	return builds, nil
}

// fetchBuildsFromPipeline fetches builds from a single pipeline with parallel pagination
func fetchBuildsFromPipeline(c *gin.Context, token, org, pipeline string, createdFrom time.Time) ([]BuildkiteBuild, error) {
	// First, fetch page 1 to check total count
	firstPageURL := fmt.Sprintf("%s/organizations/%s/pipelines/%s/builds?created_from=%s&per_page=%d&page=1",
		buildkiteBaseURL, org, pipeline, url.QueryEscape(createdFrom.Format(time.RFC3339)), buildkitePerPage)

	<-buildkiteRateLimiter.C // Rate limit
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, firstPageURL, nil)
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

	var firstPageBuilds []BuildkiteBuild
	if err := json.Unmarshal(body, &firstPageBuilds); err != nil {
		return nil, err
	}

	if len(firstPageBuilds) < buildkitePerPage {
		// Only one page
		log.Printf("[BuildKite] Total builds fetched: %d (1 page)", len(firstPageBuilds))
		return firstPageBuilds, nil
	}

	// Determine how many pages to fetch (cap at buildkiteMaxPages)
	totalPages := buildkiteMaxPages
	if len(firstPageBuilds) < buildkitePerPage {
		totalPages = 1
	}

	// Fetch remaining pages in parallel
	type pageResult struct {
		page   int
		builds []BuildkiteBuild
		err    error
	}

	results := make(chan pageResult, totalPages-1)
	var wg sync.WaitGroup

	for page := 2; page <= totalPages; page++ {
		wg.Add(1)
		go func(pageNum int) {
			defer wg.Done()

			query := url.Values{}
			query.Set("created_from", createdFrom.Format(time.RFC3339))
			query.Set("per_page", fmt.Sprintf("%d", buildkitePerPage))
			query.Set("page", fmt.Sprintf("%d", pageNum))

			pageURL := fmt.Sprintf("%s/organizations/%s/pipelines/%s/builds?%s",
				buildkiteBaseURL, org, pipeline, query.Encode())

			// Rate limit: wait for token
			<-buildkiteRateLimiter.C

			req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, pageURL, nil)
			if err != nil {
				results <- pageResult{page: pageNum, err: err}
				return
			}

			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Accept", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				results <- pageResult{page: pageNum, err: err}
				return
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)

			if resp.StatusCode != http.StatusOK {
				results <- pageResult{page: pageNum, err: fmt.Errorf("page %d: %d %s", pageNum, resp.StatusCode, string(body))}
				return
			}

			var builds []BuildkiteBuild
			if err := json.Unmarshal(body, &builds); err != nil {
				results <- pageResult{page: pageNum, err: err}
				return
			}

			results <- pageResult{page: pageNum, builds: builds}
		}(page)
	}

	// Close results channel after all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	allBuilds := make(map[int][]BuildkiteBuild)
	allBuilds[1] = firstPageBuilds

	for res := range results {
		if res.err != nil {
			log.Printf("[BuildKite] Error fetching page %d: %v", res.page, res.err)
			continue
		}
		if len(res.builds) == 0 {
			break // No more pages
		}
		allBuilds[res.page] = res.builds
	}

	// Combine all pages in order
	var combined []BuildkiteBuild
	for page := 1; page <= totalPages; page++ {
		if builds, ok := allBuilds[page]; ok {
			combined = append(combined, builds...)
		}
	}

	log.Printf("[BuildKite] Total builds fetched from %s: %d (%d pages in parallel)", pipeline, len(combined), len(allBuilds))
	return combined, nil
}

// fetchBuildsParallel fetches builds from both deployment pipelines
func fetchBuildsParallel(c *gin.Context, token, org string, createdFrom time.Time) ([]BuildkiteBuild, error) {
	pipelines := []string{
		"core-stack-deployment-pipeline",
		"core-stack-deployment-pipeline-legacy",
	}

	var allBuilds []BuildkiteBuild
	for _, pipeline := range pipelines {
		builds, err := fetchBuildsFromPipeline(c, token, org, pipeline, createdFrom)
		if err != nil {
			log.Printf("[BuildKite] Warning: Failed to fetch from %s: %v", pipeline, err)
			continue // Continue with other pipelines even if one fails
		}
		allBuilds = append(allBuilds, builds...)
	}

	log.Printf("[BuildKite] Total builds fetched from all pipelines: %d", len(allBuilds))
	return allBuilds, nil
}

// kpiBuildkiteCombinedAll returns both weekly and daily metrics in a single request
func kpiBuildkiteCombinedAll(c *gin.Context) {
	token, org, ok := buildkiteConfig()
	if !ok {
		missing := buildkiteConfigMissing()
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "BuildKite not configured",
			"missing": missing,
			"hint":    "Set BUILDKITE_TOKEN and BUILDKITE_ORG in .env",
		})
		return
	}

	// Fetch builds from last 3 months (fetch once, use for both weekly and daily)
	threeMonthsAgo := time.Now().AddDate(0, -3, 0)
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	startTime := time.Now()

	builds, err := getCachedBuilds(c, token, org, threeMonthsAgo)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch builds: " + err.Error()})
		return
	}

	fetchDuration := time.Since(startTime)
	log.Printf("[BuildKite Combined] Processing %d builds", len(builds))

	// Process data for weekly metrics
	weekDurations := make(map[string][]float64)
	weekPassed := make(map[string]int)
	weekFailed := make(map[string]int)
	weeklyDeploymentCount := 0
	weeklyPassedCount := 0
	weeklyFailedCount := 0

	// Process data for daily metrics (last 30 days only)
	dayDurations := make(map[string][]float64)
	dayPassed := make(map[string]int)
	dayFailed := make(map[string]int)
	dailyDeploymentCount := 0
	dailyPassedCount := 0
	dailyFailedCount := 0

	for _, build := range builds {
		if !isDeploymentPipeline(build) {
			continue
		}

		finishedAt, okFinish := parseTime(build.FinishedAt)
		if !okFinish {
			continue
		}

		week := weekKey(finishedAt)
		day := dayKey(finishedAt)

		// Process for weekly
		weeklyDeploymentCount++
		if build.State == "passed" {
			startedAt, okStart := parseTime(build.StartedAt)
			if okStart && finishedAt.After(startedAt) {
				durationMinutes := finishedAt.Sub(startedAt).Minutes()
				weekDurations[week] = append(weekDurations[week], durationMinutes)
			}
			weekPassed[week]++
			weeklyPassedCount++
		}
		if build.State == "failed" {
			weekFailed[week]++
			weeklyFailedCount++
		}

		// Process for daily (last 30 days only)
		if finishedAt.After(thirtyDaysAgo) {
			dailyDeploymentCount++
			if build.State == "passed" {
				startedAt, okStart := parseTime(build.StartedAt)
				if okStart && finishedAt.After(startedAt) {
					durationMinutes := finishedAt.Sub(startedAt).Minutes()
					dayDurations[day] = append(dayDurations[day], durationMinutes)
				}
				dayPassed[day]++
				dailyPassedCount++
			}
			if build.State == "failed" {
				dayFailed[day]++
				dailyFailedCount++
			}
		}
	}

	// Calculate weekly metrics
	var weeksWithDurations []string
	for w := range weekDurations {
		weeksWithDurations = append(weeksWithDurations, w)
	}
	sort.Strings(weeksWithDurations)

	weeklyAvgDurations := make([]float64, len(weeksWithDurations))
	for i, w := range weeksWithDurations {
		durations := weekDurations[w]
		if len(durations) > 0 {
			var sum float64
			for _, d := range durations {
				sum += d
			}
			weeklyAvgDurations[i] = sum / float64(len(durations))
		}
	}

	weeksMap := make(map[string]struct{})
	for w := range weekPassed {
		weeksMap[w] = struct{}{}
	}
	for w := range weekFailed {
		weeksMap[w] = struct{}{}
	}

	var weeksForFailureRate []string
	for w := range weeksMap {
		weeksForFailureRate = append(weeksForFailureRate, w)
	}
	sort.Strings(weeksForFailureRate)

	weeklyFailureRates := make([]float64, len(weeksForFailureRate))
	weeklyPassedCounts := make([]int, len(weeksForFailureRate))
	weeklyFailedCounts := make([]int, len(weeksForFailureRate))

	for i, w := range weeksForFailureRate {
		passed := weekPassed[w]
		failed := weekFailed[w]
		total := passed + failed

		weeklyPassedCounts[i] = passed
		weeklyFailedCounts[i] = failed

		if total > 0 {
			weeklyFailureRates[i] = float64(failed) / float64(total) * 100
		}
	}

	// Calculate daily metrics
	var daysWithDurations []string
	for d := range dayDurations {
		daysWithDurations = append(daysWithDurations, d)
	}
	sort.Strings(daysWithDurations)

	dailyAvgDurations := make([]float64, len(daysWithDurations))
	for i, d := range daysWithDurations {
		durations := dayDurations[d]
		if len(durations) > 0 {
			var sum float64
			for _, dur := range durations {
				sum += dur
			}
			dailyAvgDurations[i] = sum / float64(len(durations))
		}
	}

	daysMap := make(map[string]struct{})
	for d := range dayPassed {
		daysMap[d] = struct{}{}
	}
	for d := range dayFailed {
		daysMap[d] = struct{}{}
	}

	var daysForFailureRate []string
	for d := range daysMap {
		daysForFailureRate = append(daysForFailureRate, d)
	}
	sort.Strings(daysForFailureRate)

	dailyFailureRates := make([]float64, len(daysForFailureRate))
	dailyPassedCounts := make([]int, len(daysForFailureRate))
	dailyFailedCounts := make([]int, len(daysForFailureRate))

	for i, d := range daysForFailureRate {
		passed := dayPassed[d]
		failed := dayFailed[d]
		total := passed + failed

		dailyPassedCounts[i] = passed
		dailyFailedCounts[i] = failed

		if total > 0 {
			dailyFailureRates[i] = float64(failed) / float64(total) * 100
		}
	}

	log.Printf("[BuildKite Combined] Processed in %v total (weekly: %d builds, daily: %d builds)",
		time.Since(startTime), weeklyDeploymentCount, dailyDeploymentCount)

	c.JSON(http.StatusOK, gin.H{
		"weekly": gin.H{
			"deployment_time": gin.H{
				"weeks":             weeksWithDurations,
				"avg_duration_mins": weeklyAvgDurations,
			},
			"failure_rate": gin.H{
				"weeks":        weeksForFailureRate,
				"failure_rate": weeklyFailureRates,
				"passed":       weeklyPassedCounts,
				"failed":       weeklyFailedCounts,
			},
		},
		"daily": gin.H{
			"deployment_time": gin.H{
				"days":              daysWithDurations,
				"avg_duration_mins": dailyAvgDurations,
			},
			"failure_rate": gin.H{
				"days":         daysForFailureRate,
				"failure_rate": dailyFailureRates,
				"passed":       dailyPassedCounts,
				"failed":       dailyFailedCounts,
			},
		},
		"meta": gin.H{
			"total_builds":         len(builds),
			"weekly_deployments":   weeklyDeploymentCount,
			"daily_deployments":    dailyDeploymentCount,
			"date_range":           fmt.Sprintf("last 3 months (from %s)", threeMonthsAgo.Format("2006-01-02")),
			"fetch_duration_sec":   fetchDuration.Seconds(),
			"cached":               fetchDuration.Seconds() < 0.1,
			"org":                  org,
		},
	})
}

// kpiBuildkiteCombined returns both deployment time and failure rate in a single request (weekly only - DEPRECATED, use kpiBuildkiteCombinedAll)
func kpiBuildkiteCombined(c *gin.Context) {
	token, org, ok := buildkiteConfig()
	if !ok {
		missing := buildkiteConfigMissing()
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "BuildKite not configured",
			"missing": missing,
			"hint":    "Set BUILDKITE_TOKEN and BUILDKITE_ORG in .env",
		})
		return
	}

	// Fetch builds from last 3 months (only once!)
	threeMonthsAgo := time.Now().AddDate(0, -3, 0)
	startTime := time.Now()

	builds, err := fetchBuildsParallel(c, token, org, threeMonthsAgo)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch builds: " + err.Error()})
		return
	}

	fetchDuration := time.Since(startTime)
	log.Printf("[BuildKite] Fetched %d builds in %v", len(builds), fetchDuration)

	// Process data for both metrics simultaneously
	weekDurations := make(map[string][]float64)
	weekPassed := make(map[string]int)
	weekFailed := make(map[string]int)
	deploymentCount := 0
	passedCount := 0
	failedCount := 0

	for _, build := range builds {
		if !isDeploymentPipeline(build) {
			continue
		}

		finishedAt, okFinish := parseTime(build.FinishedAt)
		if !okFinish {
			continue
		}

		week := weekKey(finishedAt)
		deploymentCount++

		// For deployment time: only count passed builds
		if build.State == "passed" {
			startedAt, okStart := parseTime(build.StartedAt)
			if okStart && finishedAt.After(startedAt) {
				durationMinutes := finishedAt.Sub(startedAt).Minutes()
				weekDurations[week] = append(weekDurations[week], durationMinutes)
			}
			weekPassed[week]++
			passedCount++
		}

		// For failure rate: count passed and failed
		if build.State == "failed" {
			weekFailed[week]++
			failedCount++
		}
	}

	// Calculate metrics - separate weeks for deployment time (only successful) vs failure rate (all)
	// For deployment time: only include weeks with successful deployments
	var weeksWithDurations []string
	for w := range weekDurations {
		weeksWithDurations = append(weeksWithDurations, w)
	}
	sort.Strings(weeksWithDurations)

	avgDurations := make([]float64, len(weeksWithDurations))
	for i, w := range weeksWithDurations {
		durations := weekDurations[w]
		if len(durations) > 0 {
			var sum float64
			for _, d := range durations {
				sum += d
			}
			avgDurations[i] = sum / float64(len(durations))
		}
	}

	// For failure rate: include all weeks with any deployments
	weeksMap := make(map[string]struct{})
	for w := range weekPassed {
		weeksMap[w] = struct{}{}
	}
	for w := range weekFailed {
		weeksMap[w] = struct{}{}
	}

	var weeksForFailureRate []string
	for w := range weeksMap {
		weeksForFailureRate = append(weeksForFailureRate, w)
	}
	sort.Strings(weeksForFailureRate)

	failureRates := make([]float64, len(weeksForFailureRate))
	passedCounts := make([]int, len(weeksForFailureRate))
	failedCounts := make([]int, len(weeksForFailureRate))

	for i, w := range weeksForFailureRate {
		passed := weekPassed[w]
		failed := weekFailed[w]
		total := passed + failed

		passedCounts[i] = passed
		failedCounts[i] = failed

		if total > 0 {
			failureRates[i] = float64(failed) / float64(total) * 100
		}
	}

	log.Printf("[BuildKite] Processed %d deployment builds (%d passed, %d failed) in %v total",
		deploymentCount, passedCount, failedCount, time.Since(startTime))

	c.JSON(http.StatusOK, gin.H{
		"deployment_time": gin.H{
			"weeks":             weeksWithDurations,
			"avg_duration_mins": avgDurations,
		},
		"failure_rate": gin.H{
			"weeks":        weeksForFailureRate,
			"failure_rate": failureRates,
			"passed":       passedCounts,
			"failed":       failedCounts,
		},
		"meta": gin.H{
			"total_builds":       len(builds),
			"deployment_builds":  deploymentCount,
			"passed_builds":      passedCount,
			"failed_builds":      failedCount,
			"date_range":         fmt.Sprintf("last 3 months (from %s)", threeMonthsAgo.Format("2006-01-02")),
			"fetch_duration_sec": fetchDuration.Seconds(),
			"org":                org,
		},
	})
}

// dayKey returns YYYY-MM-DD for a given time
func dayKey(t time.Time) string {
	return t.Format("2006-01-02")
}

// kpiBuildkiteCombinedDaily returns daily deployment time and failure rate for last 30 days
func kpiBuildkiteCombinedDaily(c *gin.Context) {
	token, org, ok := buildkiteConfig()
	if !ok {
		missing := buildkiteConfigMissing()
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "BuildKite not configured",
			"missing": missing,
			"hint":    "Set BUILDKITE_TOKEN and BUILDKITE_ORG in .env",
		})
		return
	}

	// Fetch builds from last 30 days
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	startTime := time.Now()

	builds, err := fetchBuildsParallel(c, token, org, thirtyDaysAgo)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch builds: " + err.Error()})
		return
	}

	fetchDuration := time.Since(startTime)
	log.Printf("[BuildKite Daily] Fetched %d builds in %v", len(builds), fetchDuration)

	// Process data for both metrics by day
	dayDurations := make(map[string][]float64)
	dayPassed := make(map[string]int)
	dayFailed := make(map[string]int)
	deploymentCount := 0
	passedCount := 0
	failedCount := 0

	for _, build := range builds {
		if !isDeploymentPipeline(build) {
			continue
		}

		finishedAt, okFinish := parseTime(build.FinishedAt)
		if !okFinish {
			continue
		}

		day := dayKey(finishedAt)
		deploymentCount++

		// For deployment time: only count passed builds
		if build.State == "passed" {
			startedAt, okStart := parseTime(build.StartedAt)
			if okStart && finishedAt.After(startedAt) {
				durationMinutes := finishedAt.Sub(startedAt).Minutes()
				dayDurations[day] = append(dayDurations[day], durationMinutes)
			}
			dayPassed[day]++
			passedCount++
		}

		// For failure rate: count passed and failed
		if build.State == "failed" {
			dayFailed[day]++
			failedCount++
		}
	}

	// Calculate metrics - separate days for deployment time (only successful) vs failure rate (all)
	// For deployment time: only include days with successful deployments
	var daysWithDurations []string
	for d := range dayDurations {
		daysWithDurations = append(daysWithDurations, d)
	}
	sort.Strings(daysWithDurations)

	avgDurations := make([]float64, len(daysWithDurations))
	for i, d := range daysWithDurations {
		durations := dayDurations[d]
		if len(durations) > 0 {
			var sum float64
			for _, dur := range durations {
				sum += dur
			}
			avgDurations[i] = sum / float64(len(durations))
		}
	}

	// For failure rate: include all days with any deployments
	daysMap := make(map[string]struct{})
	for d := range dayPassed {
		daysMap[d] = struct{}{}
	}
	for d := range dayFailed {
		daysMap[d] = struct{}{}
	}

	var daysForFailureRate []string
	for d := range daysMap {
		daysForFailureRate = append(daysForFailureRate, d)
	}
	sort.Strings(daysForFailureRate)

	failureRates := make([]float64, len(daysForFailureRate))
	passedCounts := make([]int, len(daysForFailureRate))
	failedCounts := make([]int, len(daysForFailureRate))

	for i, d := range daysForFailureRate {
		passed := dayPassed[d]
		failed := dayFailed[d]
		total := passed + failed

		passedCounts[i] = passed
		failedCounts[i] = failed

		if total > 0 {
			failureRates[i] = float64(failed) / float64(total) * 100
		}
	}

	log.Printf("[BuildKite Daily] Processed %d deployment builds (%d passed, %d failed) in %v total",
		deploymentCount, passedCount, failedCount, time.Since(startTime))

	c.JSON(http.StatusOK, gin.H{
		"deployment_time": gin.H{
			"days":              daysWithDurations,
			"avg_duration_mins": avgDurations,
		},
		"failure_rate": gin.H{
			"days":         daysForFailureRate,
			"failure_rate": failureRates,
			"passed":       passedCounts,
			"failed":       failedCounts,
		},
		"meta": gin.H{
			"total_builds":       len(builds),
			"deployment_builds":  deploymentCount,
			"passed_builds":      passedCount,
			"failed_builds":      failedCount,
			"date_range":         fmt.Sprintf("last 30 days (from %s)", thirtyDaysAgo.Format("2006-01-02")),
			"fetch_duration_sec": fetchDuration.Seconds(),
			"org":                org,
		},
	})
}
