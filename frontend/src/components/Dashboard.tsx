import { useState, useEffect } from 'react'
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  LabelList,
} from 'recharts'

export interface EpicRow {
  epic_key: string
  summary: string
  vehicle_name?: string
  start_time: string
  finish_time: string
  build_days: number
  week: string
  type: 'Rogue' | 'MachE' | 'Other'
}

interface TimeInBuildResponse {
  weeks: string[]
  rogue: number[]
  machE: number[]
  other?: number[]
  epic_rows?: EpicRow[]
  week_labels_rogue?: Record<string, string[]>
  week_labels_mach_e?: Record<string, string[]>
  week_labels_other?: Record<string, string[]>
  meta?: {
    filter_id: string
    jql_used?: string
    epic_keys?: string[]
    epics_seen: number
    rogue_n: number
    machE_n: number
    other_n?: number
  }
}

const buildChartData = (res: TimeInBuildResponse) => {
  const weeks = Array.isArray(res.weeks) ? res.weeks : []
  const rogue = Array.isArray(res.rogue) ? res.rogue : []
  const machE = Array.isArray(res.machE) ? res.machE : []
  const other = Array.isArray(res.other) ? res.other : []
  const labelsRogue = res.week_labels_rogue ?? {}
  const labelsMachE = res.week_labels_mach_e ?? {}
  const labelsOther = res.week_labels_other ?? {}
  return weeks.map((week, i) => {
    const r = rogue[i] ?? 0
    const m = machE[i] ?? 0
    const o = other[i] ?? 0
    return {
      week,
      Rogue: r > 0 ? Math.round(r * 10) / 10 : null,
      MachE: m > 0 ? Math.round(m * 10) / 10 : null,
      Other: o > 0 ? Math.round(o * 10) / 10 : null,
      vehiclesRogue: labelsRogue[week] ?? [],
      vehiclesMachE: labelsMachE[week] ?? [],
      vehiclesOther: labelsOther[week] ?? [],
    }
  })
}

const formatDateTime = (iso: string) => {
  if (!iso) return '—'
  try {
    const d = new Date(iso)
    return d.toLocaleString(undefined, { dateStyle: 'short', timeStyle: 'short' })
  } catch {
    return iso
  }
}

const LINE_HEIGHT = 12

function renderVerticalLabel(
  props: { x?: number; y?: number; width?: number; payload?: ChartPoint; index?: number },
  vehicleKey: 'vehiclesRogue' | 'vehiclesMachE' | 'vehiclesOther',
  fill: string,
  chartData: ChartPoint[]
) {
  const { x = 0, y = 0, width = 0, payload, index } = props
  const point = payload ?? (typeof index === 'number' ? chartData[index] : undefined)
  const vehicles = point?.[vehicleKey] ?? []
  if (vehicles.length === 0) return null
  const baseX = Number(x) + Number(width) / 2 + 6
  const baseY = Number(y)
  return (
    <g>
      <text x={baseX} y={baseY} textAnchor="start" fontSize={10} fill={fill} fontWeight={500}>
        {vehicles.map((v, i) => (
          <tspan key={v} x={baseX} dy={i === 0 ? 0 : LINE_HEIGHT}>
            {v}
          </tspan>
        ))}
      </text>
    </g>
  )
}

type ChartPoint = {
  week: string
  Rogue: number | null
  MachE: number | null
  Other: number | null
  vehiclesRogue: string[]
  vehiclesMachE: string[]
  vehiclesOther: string[]
}

const Dashboard = () => {
  const [data, setData] = useState<ChartPoint[]>([])
  const [epicRows, setEpicRows] = useState<EpicRow[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [meta, setMeta] = useState<TimeInBuildResponse['meta'] | null>(null)

  // Use JIRA filter 22515 — same epics as https://appliedintuition.atlassian.net/issues/?filter=22515 (includes MCE, Rogue, etc.)
  const filterId = '22515'

  // VOS tickets KPI: tickets assigned to Vehicle OS engineers during build
  const [vosData, setVosData] = useState<{ week: string; created: number; resolved: number }[]>([])
  const [vosLoading, setVosLoading] = useState(true)
  const [vosError, setVosError] = useState<string | null>(null)
  const [vosErrorDetail, setVosErrorDetail] = useState<{ jira_response?: string; retried?: number } | null>(null)
  const [vosMeta, setVosMeta] = useState<{ jql_used?: string; issues_seen?: number; api_total?: number; capped_at?: number; warning?: string } | null>(null)

  // Build Bugs KPI: bugs found after release to calibration
  const [bugsData, setBugsData] = useState<{ week: string; created: number; resolved: number }[]>([])
  const [bugsLoading, setBugsLoading] = useState(true)
  const [bugsError, setBugsError] = useState<string | null>(null)
  const [bugsMeta, setBugsMeta] = useState<{ jql_used?: string; bugs_seen?: number; date_filter?: string; note?: string } | null>(null)

  // MTBF KPI: Mean Time Between Failure - vehicle stability issue reports
  const [mtbfData, setMtbfData] = useState<{ week: string; failures: number }[]>([])
  const [mtbfLoading, setMtbfLoading] = useState(true)
  const [mtbfError, setMtbfError] = useState<string | null>(null)
  const [mtbfMeta, setMtbfMeta] = useState<{ jql_used?: string; failures_seen?: number; date_filter?: string; note?: string; drive_hours?: string; data_available?: string } | null>(null)

  useEffect(() => {
    setLoading(true)
    setError(null)
    fetch(`/api/kpi/time-in-build?filter_id=${filterId}`)
      .then(async (r) => {
        const body = await r.json().catch(() => ({}))
        if (!r.ok) {
          const msg = (body && typeof body.error === 'string') ? body.error : `HTTP ${r.status}`
          throw new Error(msg)
        }
        return body as TimeInBuildResponse
      })
      .then((res: TimeInBuildResponse) => {
        setData(buildChartData(res ?? {}))
        setEpicRows(Array.isArray(res?.epic_rows) ? res.epic_rows : [])
        setMeta(res?.meta ?? null)
      })
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    setVosLoading(true)
    setVosError(null)
    setVosErrorDetail(null)
    fetch('/api/kpi/vos-tickets')
      .then(async (r) => {
        const body = await r.json().catch(() => ({})) as { error?: string; jira_response?: string; retried?: number; weeks?: string[]; created?: number[]; resolved?: number[]; meta?: { jql_used?: string; issues_seen?: number; api_total?: number; capped_at?: number; warning?: string } }
        if (!r.ok) {
          const msg = (body && typeof body.error === 'string') ? body.error : `HTTP ${r.status}`
          setVosErrorDetail(body && (body.jira_response != null || body.retried != null) ? { jira_response: body.jira_response, retried: body.retried } : null)
          throw new Error(msg)
        }
        return body
      })
      .then((res) => {
        const weeks = Array.isArray(res.weeks) ? res.weeks : []
        const created = Array.isArray(res.created) ? res.created : []
        const resolved = Array.isArray(res.resolved) ? res.resolved : []
        setVosData(
          weeks.map((week, i) => ({
            week,
            created: created[i] ?? 0,
            resolved: resolved[i] ?? 0,
          }))
        )
        setVosMeta(res?.meta ?? null)
      })
      .catch((e) => {
        setVosData([])
        setVosMeta(null)
        setVosError(e instanceof Error ? e.message : 'Failed to load VOS tickets')
      })
      .finally(() => setVosLoading(false))
  }, [])

  useEffect(() => {
    setBugsLoading(true)
    setBugsError(null)
    fetch('/api/kpi/build-bugs')
      .then(async (r) => {
        const body = await r.json().catch(() => ({})) as { error?: string; weeks?: string[]; created?: number[]; resolved?: number[]; meta?: { jql_used?: string; bugs_seen?: number; date_filter?: string; note?: string } }
        if (!r.ok) {
          const msg = (body && typeof body.error === 'string') ? body.error : `HTTP ${r.status}`
          throw new Error(msg)
        }
        return body
      })
      .then((res) => {
        const weeks = Array.isArray(res.weeks) ? res.weeks : []
        const created = Array.isArray(res.created) ? res.created : []
        const resolved = Array.isArray(res.resolved) ? res.resolved : []
        setBugsData(
          weeks.map((week, i) => ({
            week,
            created: created[i] ?? 0,
            resolved: resolved[i] ?? 0,
          }))
        )
        setBugsMeta(res?.meta ?? null)
      })
      .catch((e) => {
        setBugsData([])
        setBugsMeta(null)
        setBugsError(e instanceof Error ? e.message : 'Failed to load build bugs')
      })
      .finally(() => setBugsLoading(false))
  }, [])

  useEffect(() => {
    setMtbfLoading(true)
    setMtbfError(null)
    fetch('/api/kpi/mtbf')
      .then(async (r) => {
        const body = await r.json().catch(() => ({})) as { error?: string; weeks?: string[]; failures?: number[]; meta?: { jql_used?: string; failures_seen?: number; date_filter?: string; note?: string; drive_hours?: string; data_available?: string } }
        if (!r.ok) {
          const msg = (body && typeof body.error === 'string') ? body.error : `HTTP ${r.status}`
          throw new Error(msg)
        }
        return body
      })
      .then((res) => {
        const weeks = Array.isArray(res.weeks) ? res.weeks : []
        const failures = Array.isArray(res.failures) ? res.failures : []
        setMtbfData(
          weeks.map((week, i) => ({
            week,
            failures: failures[i] ?? 0,
          }))
        )
        setMtbfMeta(res?.meta ?? null)
      })
      .catch((e) => {
        setMtbfData([])
        setMtbfMeta(null)
        setMtbfError(e instanceof Error ? e.message : 'Failed to load MTBF data')
      })
      .finally(() => setMtbfLoading(false))
  }, [])

  if (loading) {
    return (
      <div className="flex-1 overflow-auto flex items-center justify-center p-8">
        <div className="animate-spin h-10 w-10 border-2 border-primary border-t-transparent rounded-full" />
        <span className="ml-3 text-gray-600">Loading KPI data…</span>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex-1 overflow-auto p-8">
        <div className="max-w-4xl">
          <h1 className="text-4xl font-bold text-gray-900 mb-6">KPI Dashboard</h1>
          <div className="bg-red-50 border border-red-200 rounded-lg p-6 text-red-800">
            <p className="font-medium">Could not load Time in Build data.</p>
            <p className="mt-1">{error}</p>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="flex-1 overflow-auto">
      <div className="p-8">
        <div className="max-w-5xl">
          <h1 className="text-4xl font-bold text-gray-900 mb-2">KPI Dashboard</h1>
          <p className="text-gray-600 mb-6">
            Time in Build: epics from{' '}
            <a
              href="https://appliedintuition.atlassian.net/issues/?filter=22515"
              target="_blank"
              rel="noopener noreferrer"
              className="text-primary underline"
            >
              JIRA filter 22515
            </a>
            . Table = finished epics (by finish time); chart = weekly averages.
          </p>

          {/* Table of every finished epic — exact data that feeds the chart */}
          {epicRows.length > 0 && (
            <div className="bg-white rounded-lg border border-gray-300 p-6 mb-6 overflow-x-auto">
              <h2 className="text-xl font-semibold text-gray-900 mb-4">Finished epics (by finish time)</h2>
              <table className="w-full text-sm border-collapse">
                <thead>
                  <tr className="border-b border-gray-200 text-left text-gray-600">
                    <th className="py-2 pr-4">Epic</th>
                    <th className="py-2 pr-4 whitespace-nowrap">Start time</th>
                    <th className="py-2 pr-4 whitespace-nowrap">Finish time</th>
                    <th className="py-2 pr-4 whitespace-nowrap">Build (days)</th>
                    <th className="py-2">Type</th>
                  </tr>
                </thead>
                <tbody>
                  {epicRows.map((row, idx) => (
                    <tr key={`${row.epic_key}-${row.type}-${idx}`} className="border-b border-gray-100 hover:bg-gray-50">
                      <td className="py-2 pr-4">
                        <a
                          href={`https://appliedintuition.atlassian.net/browse/${row.epic_key}`}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-primary hover:underline"
                        >
                          {row.epic_key}
                        </a>
                        {row.vehicle_name && (
                          <span className="ml-1 text-gray-500 font-medium">{row.vehicle_name}</span>
                        )}
                        <span className="text-gray-600 ml-1">{row.summary || '—'}</span>
                      </td>
                      <td className="py-2 pr-4 whitespace-nowrap text-gray-700">{formatDateTime(row.start_time)}</td>
                      <td className="py-2 pr-4 whitespace-nowrap text-gray-700">{formatDateTime(row.finish_time)}</td>
                      <td className="py-2 pr-4 font-mono">{row.build_days}</td>
                      <td className="py-2">
                        <span
                          className={
                            row.type === 'Rogue'
                              ? 'text-blue-600'
                              : row.type === 'MachE'
                                ? 'text-red-600'
                                : 'text-green-600'
                          }
                        >
                          {row.type}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          <div className="bg-white rounded-lg border border-gray-300 p-6 mb-6">
            <h2 className="text-xl font-semibold text-gray-900 mb-4">
              Time in Build (car in → release to fleet)
            </h2>
            <p className="text-sm text-gray-500 mb-4">
              Rogue (name contains &quot;ROG&quot;): first VBUILD In Progress → last Done. MachE (name contains &quot;MCE&quot;, excluding D-Max/DMX): epic opened → release-to-fleet Done. Other: epic created → resolved. Labels show vehicles per series, stacked vertically.
            </p>
            {data.length === 0 ? (
              <p className="text-gray-500">No data points for the selected filter yet.</p>
            ) : (
              <ResponsiveContainer width="100%" height={480}>
                <LineChart data={data} margin={{ top: 24, right: 120, left: 10, bottom: 5 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                  <XAxis dataKey="week" stroke="#6b7280" fontSize={12} />
                  <YAxis stroke="#6b7280" fontSize={12} label={{ value: 'Days', angle: -90, position: 'insideLeft' }} />
                  <Tooltip
                    contentStyle={{ backgroundColor: '#fff', border: '1px solid #e5e7eb', borderRadius: '8px' }}
                    formatter={(value: number | null) => (value != null ? [value, ''] : [])}
                    labelFormatter={(label) => `Week ${label}`}
                  />
                  <Legend />
                  <Line type="linear" dataKey="Rogue" stroke="#2563eb" strokeWidth={2} dot={{ r: 4 }} name="Rogue (days)" connectNulls={false}>
                    <LabelList
                      position="right"
                      content={(props: { x?: number; y?: number; width?: number; payload?: ChartPoint; index?: number }) => renderVerticalLabel(props, 'vehiclesRogue', '#2563eb', data)}
                    />
                  </Line>
                  <Line type="linear" dataKey="MachE" stroke="#dc2626" strokeWidth={2} dot={{ r: 4 }} name="MachE (days)" connectNulls={false}>
                    <LabelList
                      position="right"
                      content={(props: { x?: number; y?: number; width?: number; payload?: ChartPoint; index?: number }) => renderVerticalLabel(props, 'vehiclesMachE', '#dc2626', data)}
                    />
                  </Line>
                  <Line type="linear" dataKey="Other" stroke="#16a34a" strokeWidth={2} dot={{ r: 4 }} name="Other (days)" strokeDasharray="4 4" connectNulls={false}>
                    <LabelList
                      position="right"
                      content={(props: { x?: number; y?: number; width?: number; payload?: ChartPoint; index?: number }) => renderVerticalLabel(props, 'vehiclesOther', '#16a34a', data)}
                    />
                  </Line>
                </LineChart>
              </ResponsiveContainer>
            )}
            {meta && (
              <div className="mt-4 space-y-2">
                <p className="text-xs text-gray-400">
                  Filter {meta.filter_id} · {meta.epics_seen} epics · {meta.rogue_n} Rogue / {meta.machE_n} MachE / {meta.other_n ?? 0} Other points
                </p>
                {((meta.epic_keys && meta.epic_keys.length > 0) || meta.jql_used) && (
                  <details className="text-xs">
                    <summary className="cursor-pointer text-gray-500 hover:text-gray-700">Validation: JQL & epic keys</summary>
                    {meta.jql_used && (
                      <pre className="mt-2 p-2 bg-gray-100 rounded overflow-x-auto break-all" title="JQL used for epic search">
                        {meta.jql_used}
                      </pre>
                    )}
                    {meta.epic_keys && meta.epic_keys.length > 0 && (
                      <p className="mt-2 text-gray-600">
                        Epics: {meta.epic_keys.slice(0, 30).join(', ')}
                        {meta.epic_keys.length > 30 ? ` … +${meta.epic_keys.length - 30} more` : ''}
                      </p>
                    )}
                  </details>
                )}
              </div>
            )}
          </div>

          {/* Tickets assigned to Vehicle OS engineers during Build */}
          <div className="bg-white rounded-lg border border-gray-300 p-6 mb-6">
            <h2 className="text-xl font-semibold text-gray-900 mb-4">
              Tickets assigned to Vehicle OS engineers during Build
            </h2>
            <p className="text-sm text-gray-500 mb-4">
              VBUILD portfolio items assigned to the Vehicle OS integration team (okta-team-vos_si), grouped by calendar week.
            </p>
            {vosLoading ? (
              <div className="flex flex-col gap-2 py-8 text-gray-500">
                <div className="flex items-center gap-2">
                  <div className="animate-spin h-5 w-5 border-2 border-primary border-t-transparent rounded-full" />
                  <span>Loading…</span>
                </div>
                <p className="text-xs text-gray-400">VOS fetches multiple pages from JIRA; may take a few seconds or retry if rate limited.</p>
              </div>
            ) : vosError ? (
              <div className="bg-amber-50 border border-amber-200 rounded-lg p-4 text-amber-800">
                <p className="font-medium">Could not load this KPI</p>
                <p className="text-sm mt-1">{vosError}</p>
                {vosErrorDetail?.retried != null && vosErrorDetail.retried > 0 && (
                  <p className="text-xs mt-2 text-amber-700">Retried {vosErrorDetail.retried} time(s) after rate limit (429).</p>
                )}
                {vosErrorDetail?.jira_response && (
                  <details className="mt-3">
                    <summary className="cursor-pointer text-xs font-medium text-amber-700">JIRA API response</summary>
                    <pre className="mt-2 p-3 bg-amber-100/50 rounded text-xs overflow-x-auto whitespace-pre-wrap break-words max-h-48 overflow-y-auto">{vosErrorDetail.jira_response}</pre>
                  </details>
                )}
              </div>
            ) : vosData.length === 0 ? (
              <p className="text-gray-500 py-4">No data for this KPI yet.</p>
            ) : (
              <ResponsiveContainer width="100%" height={320}>
                <LineChart data={vosData} margin={{ top: 5, right: 20, left: 10, bottom: 5 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                  <XAxis dataKey="week" stroke="#6b7280" fontSize={12} />
                  <YAxis stroke="#6b7280" fontSize={12} label={{ value: 'Tickets', angle: -90, position: 'insideLeft' }} />
                  <Tooltip
                    contentStyle={{ backgroundColor: '#fff', border: '1px solid #e5e7eb', borderRadius: '8px' }}
                    labelFormatter={(label) => `Week ${label}`}
                  />
                  <Legend />
                  <Line type="linear" dataKey="created" stroke="#2563eb" strokeWidth={2} dot={{ r: 4 }} name="Tickets created" />
                  <Line type="linear" dataKey="resolved" stroke="#16a34a" strokeWidth={2} dot={{ r: 4 }} name="Tickets resolved" />
                </LineChart>
              </ResponsiveContainer>
            )}
            {vosMeta && (
              <div className="mt-4 space-y-1">
                <p className="text-xs text-gray-400">
                  {vosMeta.issues_seen ?? 0} issues fetched
                  {vosMeta.api_total != null ? ` (JIRA API total: ${vosMeta.api_total})` : ''}
                </p>
                {vosMeta.warning && (
                  <p className="text-xs text-amber-600">{vosMeta.warning}</p>
                )}
                {vosMeta.jql_used && (
                  <details className="text-xs">
                    <summary className="cursor-pointer text-gray-500 hover:text-gray-700">JQL used</summary>
                    <pre className="mt-1 p-2 bg-gray-100 rounded overflow-x-auto break-all">{vosMeta.jql_used}</pre>
                  </details>
                )}
              </div>
            )}
          </div>

          {/* KPI #4: Build Issues Caught After Release to Calibration */}
          <div className="bg-white rounded-lg border border-gray-300 p-6 mb-6">
            <h2 className="text-xl font-semibold text-gray-900 mb-4">
              Build Issues Caught After Release to Calibration
            </h2>
            <p className="text-sm text-gray-500 mb-4">
              Bugs and bug reports found in VBUILD portfolio, grouped by calendar week.
            </p>
            {bugsLoading ? (
              <div className="flex items-center gap-2 py-8 text-gray-500">
                <div className="animate-spin h-5 w-5 border-2 border-primary border-t-transparent rounded-full" />
                <span>Loading…</span>
              </div>
            ) : bugsError ? (
              <div className="bg-amber-50 border border-amber-200 rounded-lg p-4 text-amber-800">
                <p className="font-medium">Could not load this KPI</p>
                <p className="text-sm mt-1">{bugsError}</p>
              </div>
            ) : bugsData.length === 0 ? (
              <p className="text-gray-500 py-4">No bug data available yet.</p>
            ) : (
              <ResponsiveContainer width="100%" height={320}>
                <LineChart data={bugsData} margin={{ top: 5, right: 20, left: 10, bottom: 5 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                  <XAxis dataKey="week" stroke="#6b7280" fontSize={12} />
                  <YAxis stroke="#6b7280" fontSize={12} label={{ value: 'Bugs', angle: -90, position: 'insideLeft' }} />
                  <Tooltip
                    contentStyle={{ backgroundColor: '#fff', border: '1px solid #e5e7eb', borderRadius: '8px' }}
                    labelFormatter={(label) => `Week ${label}`}
                  />
                  <Legend />
                  <Line type="linear" dataKey="created" stroke="#dc2626" strokeWidth={2} dot={{ r: 4 }} name="Bugs created" />
                  <Line type="linear" dataKey="resolved" stroke="#16a34a" strokeWidth={2} dot={{ r: 4 }} name="Bugs resolved" />
                </LineChart>
              </ResponsiveContainer>
            )}
            {bugsMeta && (
              <div className="mt-4 space-y-1">
                <p className="text-xs text-gray-400">
                  {bugsMeta.bugs_seen ?? 0} bugs found
                </p>
                {bugsMeta.jql_used && (
                  <details className="text-xs">
                    <summary className="cursor-pointer text-gray-500 hover:text-gray-700">JQL used</summary>
                    <pre className="mt-1 p-2 bg-gray-100 rounded overflow-x-auto break-all">{bugsMeta.jql_used}</pre>
                  </details>
                )}
              </div>
            )}
          </div>

          {/* MTBF: Mean Time Between Failure */}
          <div className="bg-white rounded-lg border border-gray-300 p-6 mb-6">
            <h2 className="text-xl font-semibold text-gray-900 mb-4">
              Mean Time Between Failure (MTBF)
            </h2>
            <p className="text-sm text-gray-500 mb-4">
              Vehicle stability issue reports from VSTAB project, grouped by calendar week.
              {mtbfMeta?.data_available === 'failures only' && (
                <span className="ml-1 text-amber-600">Note: Drive hours data pending - currently showing failure counts only.</span>
              )}
            </p>
            {mtbfLoading ? (
              <div className="flex items-center gap-2 py-8 text-gray-500">
                <div className="animate-spin h-5 w-5 border-2 border-primary border-t-transparent rounded-full" />
                <span>Loading…</span>
              </div>
            ) : mtbfError ? (
              <div className="bg-amber-50 border border-amber-200 rounded-lg p-4 text-amber-800">
                <p className="font-medium">Could not load this KPI</p>
                <p className="text-sm mt-1">{mtbfError}</p>
              </div>
            ) : mtbfData.length === 0 ? (
              <p className="text-gray-500 py-4">No MTBF data available yet.</p>
            ) : (
              <ResponsiveContainer width="100%" height={320}>
                <LineChart data={mtbfData} margin={{ top: 5, right: 20, left: 10, bottom: 5 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                  <XAxis dataKey="week" stroke="#6b7280" fontSize={12} />
                  <YAxis stroke="#6b7280" fontSize={12} label={{ value: 'Failures', angle: -90, position: 'insideLeft' }} />
                  <Tooltip
                    contentStyle={{ backgroundColor: '#fff', border: '1px solid #e5e7eb', borderRadius: '8px' }}
                    labelFormatter={(label) => `Week ${label}`}
                  />
                  <Legend />
                  <Line type="linear" dataKey="failures" stroke="#f59e0b" strokeWidth={2} dot={{ r: 4 }} name="Stability failures" />
                </LineChart>
              </ResponsiveContainer>
            )}
            {mtbfMeta && (
              <div className="mt-4 space-y-1">
                <p className="text-xs text-gray-400">
                  {mtbfMeta.failures_seen ?? 0} failures found
                  {mtbfMeta.date_filter && ` · ${mtbfMeta.date_filter}`}
                </p>
                {mtbfMeta.note && (
                  <p className="text-xs text-blue-600">{mtbfMeta.note}</p>
                )}
                {mtbfMeta.jql_used && (
                  <details className="text-xs">
                    <summary className="cursor-pointer text-gray-500 hover:text-gray-700">JQL used</summary>
                    <pre className="mt-1 p-2 bg-gray-100 rounded overflow-x-auto break-all">{mtbfMeta.jql_used}</pre>
                  </details>
                )}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

export default Dashboard
