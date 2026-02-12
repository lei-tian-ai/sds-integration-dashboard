const About = () => {
  return (
    <div className="flex-1 overflow-auto">
      <div className="p-8">
        <div className="max-w-4xl">
          <h1 className="text-4xl font-bold text-gray-900 mb-6">About This Dashboard</h1>

          <div className="bg-white rounded-lg border border-gray-300 p-6 mb-6">
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">Overview</h2>
            <p className="text-gray-700 mb-3">
              This dashboard tracks key performance indicators (KPIs) for the vehicle build process at Applied Intuition.
              It aggregates data from JIRA and Fleetio to provide visibility into build timelines, engineering workload,
              quality issues, and vehicle reliability.
            </p>
            <p className="text-gray-700">
              All metrics are calculated weekly using ISO week numbers (e.g., 2025-W06) and displayed as time-series charts.
            </p>
          </div>

          <div className="space-y-6">
            {/* Metric 1: Time in Build */}
            <div className="bg-white rounded-lg border border-gray-300 p-6">
              <h3 className="text-xl font-semibold text-gray-900 mb-3">
                Metric #1: Time in Build (Car In → Release to Fleet)
              </h3>

              <div className="space-y-3 text-sm text-gray-700">
                <div>
                  <span className="font-medium">What it measures:</span> Average days from vehicle intake to fleet release, broken down by vehicle line (Rogue, MachE, Other).
                </div>

                <div>
                  <span className="font-medium">Data source:</span> JIRA epics from{' '}
                  <a
                    href="https://appliedintuition.atlassian.net/issues/?filter=22515"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-primary underline"
                  >
                    filter 22515
                  </a>
                  {' '}(vehicle build epics)
                </div>

                <div>
                  <span className="font-medium">How it's calculated:</span>
                  <ul className="list-disc ml-6 mt-2 space-y-1">
                    <li>
                      <strong>Rogue</strong> (epic name contains "ROG"): Days from first VBUILD ticket transitioned to "In Progress" → last VBUILD ticket moved to "Done"
                    </li>
                    <li>
                      <strong>MachE</strong> (epic name contains "MCE", excluding D-Max/DMX): Days from epic created → "release to fleet" child ticket moved to "Done"
                    </li>
                    <li>
                      <strong>Other</strong>: Days from epic created → epic resolved
                    </li>
                  </ul>
                </div>

                <div>
                  <span className="font-medium">Aggregation:</span> Weekly average of all epics completed that week (grouped by ISO week of completion)
                </div>

                <div>
                  <span className="font-medium">Time range:</span> Last 730 days (2 years) of epic data to capture closed epics for trend analysis
                </div>

                <div>
                  <span className="font-medium">API endpoint:</span> <code className="bg-gray-100 px-2 py-1 rounded text-xs">GET /api/kpi/time-in-build?filter_id=22515</code>
                </div>
              </div>
            </div>

            {/* Metric 2: VOS Tickets */}
            <div className="bg-white rounded-lg border border-gray-300 p-6">
              <h3 className="text-xl font-semibold text-gray-900 mb-3">
                Metric #2: Tickets Assigned to Vehicle OS Engineers During Build
              </h3>

              <div className="space-y-3 text-sm text-gray-700">
                <div>
                  <span className="font-medium">What it measures:</span> Weekly ticket volume (created vs. resolved) for Vehicle OS integration team during vehicle builds.
                </div>

                <div>
                  <span className="font-medium">Data source:</span> JIRA tickets matching VOS team filter
                </div>

                <div>
                  <span className="font-medium">JQL query:</span>
                  <pre className="mt-2 p-3 bg-gray-100 rounded text-xs overflow-x-auto">
                    project in (10525) AND 'issue' in portfolioChildIssuesOf(VBUILD-8121) and assignee in membersOf("okta-team-vos_si")
                  </pre>
                </div>

                <div>
                  <span className="font-medium">How it's calculated:</span>
                  <ul className="list-disc ml-6 mt-2 space-y-1">
                    <li><strong>Created:</strong> Tickets created during each calendar week</li>
                    <li><strong>Resolved:</strong> Tickets with resolutiondate during each calendar week</li>
                  </ul>
                </div>

                <div>
                  <span className="font-medium">Time range:</span> Last 2 months (8-9 weeks)
                </div>

                <div>
                  <span className="font-medium">Performance optimization:</span> Uses parallel week-by-week queries to avoid JIRA API pagination issues and rate limits
                </div>

                <div>
                  <span className="font-medium">API endpoint:</span> <code className="bg-gray-100 px-2 py-1 rounded text-xs">GET /api/kpi/vos-tickets</code>
                </div>
              </div>
            </div>

            {/* Metric 3: Build Bugs */}
            <div className="bg-white rounded-lg border border-gray-300 p-6">
              <h3 className="text-xl font-semibold text-gray-900 mb-3">
                Metric #3: Build Issues Caught After Release to Calibration
              </h3>

              <div className="space-y-3 text-sm text-gray-700">
                <div>
                  <span className="font-medium">What it measures:</span> Bug reports discovered in VBUILD portfolio after vehicle release, tracked weekly.
                </div>

                <div>
                  <span className="font-medium">Data source:</span> JIRA bugs in VBUILD portfolio
                </div>

                <div>
                  <span className="font-medium">JQL query:</span>
                  <pre className="mt-2 p-3 bg-gray-100 rounded text-xs overflow-x-auto">
                    project in (10525) AND 'issue' in portfolioChildIssuesOf(VBUILD-8121) AND type in ("Bug", "Bug Report")
                  </pre>
                </div>

                <div>
                  <span className="font-medium">How it's calculated:</span>
                  <ul className="list-disc ml-6 mt-2 space-y-1">
                    <li><strong>Created:</strong> Bug tickets created during each calendar week</li>
                    <li><strong>Resolved:</strong> Bug tickets with resolutiondate during each calendar week</li>
                  </ul>
                </div>

                <div>
                  <span className="font-medium">Time range:</span> Last 2 months
                </div>

                <div>
                  <span className="font-medium">Purpose:</span> Track quality issues to identify trends and improve pre-release testing
                </div>

                <div>
                  <span className="font-medium">API endpoint:</span> <code className="bg-gray-100 px-2 py-1 rounded text-xs">GET /api/kpi/build-bugs</code>
                </div>
              </div>
            </div>

            {/* Metric 4: MTBF */}
            <div className="bg-white rounded-lg border border-gray-300 p-6">
              <h3 className="text-xl font-semibold text-gray-900 mb-3">
                Metric #5: Mean Time Between Failure (MTBF)
              </h3>

              <div className="space-y-3 text-sm text-gray-700">
                <div>
                  <span className="font-medium">What it measures:</span> Vehicle reliability metric tracking stability failures over time (intended: failures per operation hour).
                </div>

                <div className="bg-amber-50 border border-amber-200 rounded p-3">
                  <span className="font-medium text-amber-800">⚠️ Status: Incomplete</span>
                  <p className="text-amber-700 mt-1">
                    Currently tracking failure counts only. Operation hours data pending from Fleetio meter entries API.
                  </p>
                </div>

                <div>
                  <span className="font-medium">Data source (current):</span> JIRA Vehicle Stability Issue Reports
                </div>

                <div>
                  <span className="font-medium">JQL query:</span>
                  <pre className="mt-2 p-3 bg-gray-100 rounded text-xs overflow-x-auto">
                    project = VSTAB AND type = "Vehicle Stability Issue Report" AND component = "On Road Dev"
                  </pre>
                </div>

                <div>
                  <span className="font-medium">Data source (needed):</span> Fleetio meter entries for vehicle operation hours
                  <ul className="list-disc ml-6 mt-2 space-y-1">
                    <li>Endpoint: <code className="bg-gray-100 px-1 rounded text-xs">GET /meter_entries</code></li>
                    <li>Need to aggregate engine hours or primary meter readings per vehicle per week</li>
                    <li>Match meter data with failure data by calendar week</li>
                  </ul>
                </div>

                <div>
                  <span className="font-medium">Intended calculation:</span> MTBF = total_operation_hours / failure_count (higher is better)
                </div>

                <div>
                  <span className="font-medium">Time range:</span> Last 3 months
                </div>

                <div>
                  <span className="font-medium">API endpoint:</span> <code className="bg-gray-100 px-2 py-1 rounded text-xs">GET /api/kpi/mtbf</code>
                </div>
              </div>
            </div>
            {/* Metric 6: Average Deployment Time */}
            <div className="bg-white rounded-lg border border-gray-300 p-6">
              <h3 className="text-xl font-semibold text-gray-900 mb-3">
                Metric #6: Average Deployment Time
              </h3>

              <div className="space-y-3 text-sm text-gray-700">
                <div>
                  <span className="font-medium">What it measures:</span> Average time (in minutes) from build start to completion for deployment pipelines.
                </div>

                <div>
                  <span className="font-medium">Data source:</span> BuildKite REST API
                </div>

                <div>
                  <span className="font-medium">API endpoint used:</span>
                  <pre className="mt-2 p-3 bg-gray-100 rounded text-xs overflow-x-auto">
                    GET https://api.buildkite.com/v2/organizations/&#123;org&#125;/builds
                  </pre>
                </div>

                <div>
                  <span className="font-medium">How it's calculated:</span>
                  <ul className="list-disc ml-6 mt-2 space-y-1">
                    <li>Fetch builds from last 3 months</li>
                    <li>Filter by deployment pipelines (slug contains "deploy-" or branch is "main"/"production")</li>
                    <li>Only include passed builds (state = "passed")</li>
                    <li>Duration = finished_at - started_at (in minutes)</li>
                    <li>Group by ISO week (based on finished_at)</li>
                    <li>Calculate average duration per week</li>
                  </ul>
                </div>

                <div>
                  <span className="font-medium">Time range:</span> Last 3 months
                </div>

                <div>
                  <span className="font-medium">Authentication:</span> Bearer token (BUILDKITE_TOKEN env var)
                </div>

                <div>
                  <span className="font-medium">Backend endpoint:</span> <code className="bg-gray-100 px-2 py-1 rounded text-xs">GET /api/kpi/buildkite-deployment-time</code>
                </div>
              </div>
            </div>

            {/* Metric 7: Deployment Failure Rate */}
            <div className="bg-white rounded-lg border border-gray-300 p-6">
              <h3 className="text-xl font-semibold text-gray-900 mb-3">
                Metric #7: Deployment Failure Rate
              </h3>

              <div className="space-y-3 text-sm text-gray-700">
                <div>
                  <span className="font-medium">What it measures:</span> Percentage of failed deployments per week (reliability metric).
                </div>

                <div>
                  <span className="font-medium">Data source:</span> BuildKite REST API
                </div>

                <div>
                  <span className="font-medium">How it's calculated:</span>
                  <ul className="list-disc ml-6 mt-2 space-y-1">
                    <li>Fetch builds from last 3 months</li>
                    <li>Filter by deployment pipelines</li>
                    <li>Count passed (state = "passed") and failed (state = "failed") builds per week</li>
                    <li>Failure rate = (failed / (passed + failed)) × 100</li>
                  </ul>
                </div>

                <div>
                  <span className="font-medium">Chart displays:</span>
                  <ul className="list-disc ml-6 mt-2 space-y-1">
                    <li><strong>Failure %:</strong> Percentage of failed deployments</li>
                    <li><strong>Passed:</strong> Count of successful deployments</li>
                    <li><strong>Failed:</strong> Count of failed deployments</li>
                  </ul>
                </div>

                <div>
                  <span className="font-medium">Time range:</span> Last 3 months
                </div>

                <div>
                  <span className="font-medium">Deployment detection:</span> Customizable in buildkite.go (currently: pipeline slug starts with "deploy-" OR branch is "main"/"production")
                </div>

                <div>
                  <span className="font-medium">Backend endpoint:</span> <code className="bg-gray-100 px-2 py-1 rounded text-xs">GET /api/kpi/buildkite-deployment-failure-rate</code>
                </div>
              </div>
            </div>
          </div>

          {/* Technical Details */}
          <div className="bg-white rounded-lg border border-gray-300 p-6 mt-6">
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">Technical Details</h2>

            <div className="space-y-4 text-sm text-gray-700">
              <div>
                <span className="font-medium">Backend:</span> Go + Gin web framework
              </div>

              <div>
                <span className="font-medium">Frontend:</span> React + TypeScript + Tailwind CSS + Recharts
              </div>

              <div>
                <span className="font-medium">JIRA Authentication:</span> Basic auth with email + API token (configured via JIRA_DOMAIN, JIRA_EMAIL, JIRA_API_TOKEN env vars)
              </div>

              <div>
                <span className="font-medium">Fleetio Authentication:</span> Account token + API key (configured via FLEETIO_ACCOUNT_TOKEN, FLEETIO_API_KEY env vars)
              </div>

              <div>
                <span className="font-medium">BuildKite Authentication:</span> Bearer token (configured via BUILDKITE_TOKEN, BUILDKITE_ORG env vars)
              </div>

              <div>
                <span className="font-medium">Date aggregation:</span> ISO 8601 week numbers (year-W##, e.g., 2025-W06)
              </div>

              <div>
                <span className="font-medium">Changelog parsing:</span> JIRA issue changelogs are parsed to extract first transition timestamps (e.g., "In Progress", "Done")
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

export default About
