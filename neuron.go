package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// TODO: Replace with actual Neuron API base URL once discovered
const neuronBaseURL = "https://neuron.oci.applied.dev"

func neuronConfig() (baseURL, token string, ok bool) {
	baseURL = strings.TrimSpace(os.Getenv("NEURON_API_URL"))
	if baseURL == "" {
		baseURL = neuronBaseURL
	}
	token = strings.TrimSpace(os.Getenv("NEURON_API_TOKEN"))
	if token == "" {
		return baseURL, "", false
	}
	return baseURL, token, true
}

func neuronConfigMissing() []string {
	var missing []string
	if strings.TrimSpace(os.Getenv("NEURON_API_TOKEN")) == "" {
		missing = append(missing, "NEURON_API_TOKEN")
	}
	return missing
}

// GET /api/neuron/vehicle-hours - get vehicle operation hours
// TODO: Update path, query params, and response parsing once API is discovered
func neuronVehicleHours(c *gin.Context) {
	baseURL, token, ok := neuronConfig()
	if !ok {
		missing := neuronConfigMissing()
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Neuron not configured",
			"missing": missing,
			"hint":    "Set NEURON_API_TOKEN in .env or environment. Use docs/neuron-api-discovery.md to find API details.",
		})
		return
	}

	// TODO: Replace with actual API path once discovered (see docs/neuron-api-discovery.md)
	// Example paths to try:
	// - /api/v1/metrics/vehicle-hours
	// - /api/vehicles/operation-hours
	// - /validation_toolset/api/metrics
	path := "/api/v1/metrics/vehicle-hours" // PLACEHOLDER - replace with actual path

	// TODO: Add query params based on dashboard URL pattern
	project := c.DefaultQuery("project", "Default")
	workspace := c.DefaultQuery("workspace", "")
	startDate := c.Query("start_date") // ISO 8601 format
	endDate := c.Query("end_date")

	// Build request URL
	reqURL := baseURL + path
	if project != "" || workspace != "" || startDate != "" || endDate != "" {
		reqURL += "?"
		params := []string{}
		if project != "" {
			params = append(params, "project="+project)
		}
		if workspace != "" {
			params = append(params, "workspace="+workspace)
		}
		if startDate != "" {
			params = append(params, "start_date="+startDate)
		}
		if endDate != "" {
			params = append(params, "end_date="+endDate)
		}
		reqURL += strings.Join(params, "&")
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, reqURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// TODO: Update auth header based on what you find in DevTools
	// Common patterns:
	// - Authorization: Bearer <token>
	// - X-API-Key: <token>
	// - Cookie: session=<token>
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Neuron request failed: " + err.Error()})
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, gin.H{
			"error":  fmt.Sprintf("Neuron API returned %d", resp.StatusCode),
			"detail": string(body),
			"hint":   "API endpoint may be incorrect. Check docs/neuron-api-discovery.md to find correct endpoint.",
		})
		return
	}

	// TODO: Update response parsing based on actual API response format
	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "invalid Neuron response: " + err.Error(),
			"hint":  "Response format may differ from expected. Check raw response in DevTools.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   data,
		"source": "neuron",
	})
}

// Example response structure (TODO: update based on actual API)
// type NeuronVehicleMetrics struct {
// 	Vehicles []struct {
// 		ID              string  `json:"id"`
// 		Name            string  `json:"name"`
// 		OperationHours  float64 `json:"operation_hours"`
// 		DriveHours      float64 `json:"drive_hours"`
// 		Date            string  `json:"date"`
// 	} `json:"vehicles"`
// }
