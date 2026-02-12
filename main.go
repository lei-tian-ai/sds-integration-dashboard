package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

//go:generate sh -c "cd frontend && npm install && npm run build"
//go:embed frontend/dist
var frontendFS embed.FS

type Response struct {
	Message string `json:"message"`
}

func main() {
	// Load .env from project root (no-op if file missing; env vars already set take precedence)
	_ = godotenv.Load()

	r := gin.Default()

	// API routes
	api := r.Group("/api")
	{
		api.GET("/hello", func(c *gin.Context) {
			c.JSON(http.StatusOK, Response{
				Message: "Hello from Go backend!",
			})
		})
		api.GET("/jira/search", jiraSearch)
		api.GET("/kpi/time-in-build", kpiTimeInBuild)
		api.GET("/kpi/debug-epic", kpiDebugEpic)
		api.GET("/kpi/vos-tickets", kpiVOSTickets)
		api.GET("/kpi/build-bugs", kpiBuildBugs)
		api.GET("/kpi/mtbf", kpiMTBF)
		api.GET("/fleetio/me", fleetioMe)
		api.GET("/fleetio/vehicles", fleetioVehicles)
		api.GET("/kpi/buildkite-deployment-time", kpiBuildkiteDeploymentTime)
		api.GET("/kpi/buildkite-deployment-failure-rate", kpiBuildkiteDeploymentFailureRate)
		api.GET("/kpi/buildkite-combined", kpiBuildkiteCombined)                 // Optimized: both metrics in one call (weekly, 3 months) - DEPRECATED
		api.GET("/kpi/buildkite-combined-daily", kpiBuildkiteCombinedDaily)      // Daily metrics (last 30 days) - DEPRECATED
		api.GET("/kpi/buildkite-combined-all", kpiBuildkiteCombinedAll)          // Optimized: weekly + daily in one call with caching
	}

	// Serve embedded frontend in production, or proxy to Vite in dev
	if os.Getenv("ENV") == "dev" {
		// In dev mode, frontend runs separately on Vite
		log.Println("Running in dev mode - frontend should be served by Vite on :3000")
	} else {
		// Serve embedded frontend
		distFS, err := fs.Sub(frontendFS, "frontend/dist")
		if err != nil {
			log.Fatal(err)
		}
		r.NoRoute(func(c *gin.Context) {
			c.FileFromFS(c.Request.URL.Path, http.FS(distFS))
		})
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	log.Printf("Server starting on port %s\n", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
