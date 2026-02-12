.PHONY: install deps backend frontend run build deploy clean check-jira check-fleetio

install:
	cd frontend && npm install

deps:
	go mod download
	cd frontend && npm install

backend:
	ENV=dev go run .

frontend:
	cd frontend && npm run dev

build-frontend:
	cd frontend && npm run build

run: deps
	@echo "Starting Go backend and React frontend in dev mode..."
	@echo "Backend will run on http://localhost:8082"
	@echo "Frontend will run on http://localhost:3000"
	@make -j2 backend frontend

build: deps
	go generate
	go build -o app .

deploy:
	apps-platform app deploy --no-build

clean:
	cd frontend && rm -rf node_modules dist
	rm -f app
	go clean

# Requires backend running (make backend) and JIRA_DOMAIN, JIRA_EMAIL, JIRA_API_TOKEN set
check-jira:
	@curl -sf -o /dev/null "http://localhost:8082/api/hello" || (echo "Backend not running. Start it in another terminal: make backend"; exit 1)
	@echo "=== 1. Default search (recent issues) ==="
	@curl -s "http://localhost:8082/api/jira/search" | head -c 500
	@echo ""
	@echo "=== 2. With maxResults=5 ==="
	@curl -s "http://localhost:8082/api/jira/search?maxResults=5" | head -c 500
	@echo ""
	@echo "=== 3. HTTP status ==="
	@curl -s -o /dev/null -w "GET /api/jira/search → %{http_code}\n" "http://localhost:8082/api/jira/search"

# Requires backend running and FLEETIO_ACCOUNT_TOKEN, FLEETIO_API_KEY set (e.g. in .env)
check-fleetio:
	@curl -sf -o /dev/null "http://localhost:8082/api/hello" || (echo "Backend not running. Start it in another terminal: make backend"; exit 1)
	@echo "=== Fleetio /me (auth check) ==="
	@curl -s "http://localhost:8082/api/fleetio/me" | head -c 400
	@echo ""
	@echo "=== Fleetio /vehicles ==="
	@curl -s "http://localhost:8082/api/fleetio/vehicles?per_page=2" | head -c 500
	@echo ""
	@echo "=== HTTP status ==="
	@curl -s -o /dev/null -w "GET /api/fleetio/me → %{http_code}\n" "http://localhost:8082/api/fleetio/me"
	@curl -s -o /dev/null -w "GET /api/fleetio/vehicles → %{http_code}\n" "http://localhost:8082/api/fleetio/vehicles"

# Debug a single epic (e.g. VBUILD-5762). Requires backend running and JIRA env set.
# Usage: make debug-epic [EPIC=VBUILD-5762]
debug-epic:
	@curl -sf -o /dev/null "http://localhost:8082/api/hello" || (echo "Backend not running. Start it: make backend"; exit 1)
	@echo "=== Debug epic: $${EPIC:-VBUILD-5762} ==="
	@curl -s "http://localhost:8082/api/kpi/debug-epic?epic=$${EPIC:-VBUILD-5762}" | python3 -m json.tool
