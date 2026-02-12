/**
 * Time in Build KPI – React component
 * Fetches GET /api/kpi/time-in-build?filter_id=22515 and renders chart + table.
 * Dependencies: react, recharts. Optional: Tailwind (or replace className with your CSS).
 */
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

export interface TimeInBuildResponse {
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

type ChartPoint = {
  week: string
  Rogue: number | null
  MachE: number | null
  Other: number | null
  vehiclesRogue: string[]
  vehiclesMachE: string[]
  vehiclesOther: string[]
}

const buildChartData = (res: TimeInBuildResponse): ChartPoint[] => {
  const weeks = Array.isArray(res.weeks) ? res.weeks : []
  const rogue = Array.isArray(res.rogue) ? res.rogue : []
  const machE = Array.isArray(res.machE) ? res.machE : []
  const other = Array.isArray(res.other) ? res.other : []
  const labelsRogue = res.week_labels_rogue ?? {}
  const labelsMachE = res.week_labels_mach_e ?? {}
  const labelsOther = res.week_labels_other ?? {}
  return weeks.map((week, i) => ({
    week,
    Rogue: (rogue[i] ?? 0) > 0 ? Math.round(rogue[i] * 10) / 10 : null,
    MachE: (machE[i] ?? 0) > 0 ? Math.round(machE[i] * 10) / 10 : null,
    Other: (other[i] ?? 0) > 0 ? Math.round(other[i] * 10) / 10 : null,
    vehiclesRogue: labelsRogue[week] ?? [],
    vehiclesMachE: labelsMachE[week] ?? [],
    vehiclesOther: labelsOther[week] ?? [],
  }))
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

export interface TimeInBuildSectionProps {
  /** Base URL for API (e.g. '' for same origin, or 'https://your-api.com') */
  apiBaseUrl?: string
  /** JIRA filter ID (default 22515) */
  filterId?: string
  /** JIRA base URL for issue links (e.g. https://appliedintuition.atlassian.net) */
  jiraBaseUrl?: string
}

export function TimeInBuildSection({
  apiBaseUrl = '',
  filterId = '22515',
  jiraBaseUrl = 'https://appliedintuition.atlassian.net',
}: TimeInBuildSectionProps) {
  const [data, setData] = useState<ChartPoint[]>([])
  const [epicRows, setEpicRows] = useState<EpicRow[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [meta, setMeta] = useState<TimeInBuildResponse['meta'] | null>(null)

  useEffect(() => {
    setLoading(true)
    setError(null)
    fetch(`${apiBaseUrl}/api/kpi/time-in-build?filter_id=${filterId}`)
      .then(async (r) => {
        const body = await r.json().catch(() => ({}))
        if (!r.ok) {
          const msg = (body && typeof body.error === 'string') ? body.error : `HTTP ${r.status}`
          throw new Error(msg)
        }
        return body as TimeInBuildResponse
      })
      .then((res) => {
        setData(buildChartData(res ?? {}))
        setEpicRows(Array.isArray(res?.epic_rows) ? res.epic_rows : [])
        setMeta(res?.meta ?? null)
      })
      .catch((e) => setError(e instanceof Error ? e.message : 'Failed to load'))
      .finally(() => setLoading(false))
  }, [apiBaseUrl, filterId])

  if (loading) {
    return (
      <div className="flex items-center gap-2 py-8 text-gray-500">
        <div className="animate-spin h-8 w-8 border-2 border-blue-500 border-t-transparent rounded-full" />
        <span>Loading Time in Build…</span>
      </div>
    )
  }

  if (error) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-lg p-4 text-red-800">
        <p className="font-medium">Could not load Time in Build data.</p>
        <p className="text-sm mt-1">{error}</p>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {epicRows.length > 0 && (
        <div className="bg-white rounded-lg border border-gray-300 p-6 overflow-x-auto">
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
                      href={`${jiraBaseUrl}/browse/${row.epic_key}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-blue-600 hover:underline"
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

      <div className="bg-white rounded-lg border border-gray-300 p-6">
        <h2 className="text-xl font-semibold text-gray-900 mb-4">Time in Build (car in → release to fleet)</h2>
        <p className="text-sm text-gray-500 mb-4">
          Rogue (ROG): epic created → resolved. MachE (MCE, excl. D-Max/DMX): epic created → resolved. Other: epic created → resolved. Labels = vehicles per series.
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
              <Line type="monotone" dataKey="Rogue" stroke="#2563eb" strokeWidth={2} dot={{ r: 4 }} name="Rogue (days)" connectNulls={false}>
                <LabelList
                  position="right"
                  content={(props) => renderVerticalLabel(props, 'vehiclesRogue', '#2563eb', data)}
                />
              </Line>
              <Line type="monotone" dataKey="MachE" stroke="#dc2626" strokeWidth={2} dot={{ r: 4 }} name="MachE (days)" connectNulls={false}>
                <LabelList
                  position="right"
                  content={(props) => renderVerticalLabel(props, 'vehiclesMachE', '#dc2626', data)}
                />
              </Line>
              <Line type="monotone" dataKey="Other" stroke="#16a34a" strokeWidth={2} dot={{ r: 4 }} name="Other (days)" strokeDasharray="4 4" connectNulls={false}>
                <LabelList
                  position="right"
                  content={(props) => renderVerticalLabel(props, 'vehiclesOther', '#16a34a', data)}
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
            {(meta.epic_keys?.length || meta.jql_used) && (
              <details className="text-xs">
                <summary className="cursor-pointer text-gray-500 hover:text-gray-700">JQL & epic keys</summary>
                {meta.jql_used && (
                  <pre className="mt-2 p-2 bg-gray-100 rounded overflow-x-auto break-all">{meta.jql_used}</pre>
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
    </div>
  )
}
