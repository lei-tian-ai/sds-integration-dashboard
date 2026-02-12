package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const fleetioBaseURL = "https://secure.fleetio.com/api/v1"

func fleetioConfig() (accountToken, apiKey string, ok bool) {
	accountToken = strings.TrimSpace(os.Getenv("FLEETIO_ACCOUNT_TOKEN"))
	apiKey = strings.TrimSpace(os.Getenv("FLEETIO_API_KEY"))
	if accountToken == "" || apiKey == "" {
		return "", "", false
	}
	return accountToken, apiKey, true
}

func fleetioConfigMissing() []string {
	var missing []string
	if strings.TrimSpace(os.Getenv("FLEETIO_ACCOUNT_TOKEN")) == "" {
		missing = append(missing, "FLEETIO_ACCOUNT_TOKEN")
	}
	if strings.TrimSpace(os.Getenv("FLEETIO_API_KEY")) == "" {
		missing = append(missing, "FLEETIO_API_KEY")
	}
	return missing
}

// GET /api/fleetio/me – current user (test auth)
func fleetioMe(c *gin.Context) {
	accountToken, apiKey, ok := fleetioConfig()
	if !ok {
		missing := fleetioConfigMissing()
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Fleetio not configured",
			"missing": missing,
			"hint":    "Set FLEETIO_ACCOUNT_TOKEN and FLEETIO_API_KEY in .env or environment",
		})
		return
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, fleetioBaseURL+"/users/me", nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	req.Header.Set("Authorization", "Token "+apiKey)
	req.Header.Set("Account-Token", accountToken)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Fleetio request failed: " + err.Error()})
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, gin.H{
			"error":  fmt.Sprintf("Fleetio API returned %d", resp.StatusCode),
			"detail": string(body),
		})
		return
	}

	var user map[string]interface{}
	if err := json.Unmarshal(body, &user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid Fleetio response"})
		return
	}
	c.JSON(http.StatusOK, user)
}

// GET /api/fleetio/vehicles – list vehicles (paginated)
func fleetioVehicles(c *gin.Context) {
	accountToken, apiKey, ok := fleetioConfig()
	if !ok {
		missing := fleetioConfigMissing()
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Fleetio not configured",
			"missing": missing,
			"hint":    "Set FLEETIO_ACCOUNT_TOKEN and FLEETIO_API_KEY in .env or environment",
		})
		return
	}

	perPage := c.DefaultQuery("per_page", "25")
	page := c.DefaultQuery("page", "1")
	if _, err := strconv.Atoi(perPage); err != nil {
		perPage = "25"
	}
	if _, err := strconv.Atoi(page); err != nil {
		page = "1"
	}

	path := "/vehicles?per_page=" + perPage + "&page=" + page
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, fleetioBaseURL+path, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	req.Header.Set("Authorization", "Token "+apiKey)
	req.Header.Set("Account-Token", accountToken)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Fleetio request failed: " + err.Error()})
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, gin.H{
			"error":  fmt.Sprintf("Fleetio API returned %d", resp.StatusCode),
			"detail": string(body),
		})
		return
	}

	var data []map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid Fleetio response: " + err.Error()})
		return
	}

	// Pass through pagination headers if present
	totalCount := resp.Header.Get("X-Pagination-Total-Count")
	totalPages := resp.Header.Get("X-Pagination-Total-Pages")
	currentPage := resp.Header.Get("X-Pagination-Current-Page")

	c.JSON(http.StatusOK, gin.H{
		"vehicles":     data,
		"total_count":  totalCount,
		"total_pages":  totalPages,
		"current_page": currentPage,
	})
}
