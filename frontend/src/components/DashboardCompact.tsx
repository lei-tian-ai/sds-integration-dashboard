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
  ReferenceLine,
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
  meta?: any
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

const formatDateOnly = (iso: string) => {
  if (!iso) return '—'
  try {
    const d = new Date(iso)
    const month = String(d.getMonth() + 1).padStart(2, '0')
    const day = String(d.getDate()).padStart(2, '0')
    const year = String(d.getFullYear()).slice(-2)
    return `${month}/${day}/${year}`
  } catch {
    return iso
  }
}

const LINE_HEIGHT = 10

type ChartPoint = {
  week: string
  Rogue: number | null
  MachE: number | null
  Other: number | null
  vehiclesRogue: string[]
  vehiclesMachE: string[]
  vehiclesOther: string[]
}

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
  const baseX = Number(x) + Number(width) / 2 + 4
  const baseY = Number(y)
  return (
    <g>
      <text x={baseX} y={baseY} textAnchor="start" fontSize={8} fill={fill} fontWeight={500}>
        {vehicles.map((v, i) => (
          <tspan key={v} x={baseX} dy={i === 0 ? 0 : LINE_HEIGHT}>
            {v}
          </tspan>
        ))}
      </text>
    </g>
  )
}

const DashboardCompact = () => {
  const [epicRows, setEpicRows] = useState<EpicRow[]>([])
  const [chartData, setChartData] = useState<ChartPoint[]>([])
  const [loading, setLoading] = useState(true)
  const [vosData, setVosData] = useState<any[]>([])
  const [bugsData, setBugsData] = useState<any[]>([])
  const [mtbfData, setMtbfData] = useState<any[]>([])
  const [deployTimeData, setDeployTimeData] = useState<any[]>([])
  const [deployFailureData, setDeployFailureData] = useState<any[]>([])

  useEffect(() => {
    // Fetch Time in Build
    fetch('/api/kpi/time-in-build?filter_id=22515')
      .then((r) => r.json())
      .then((res: TimeInBuildResponse) => {
        setEpicRows(res.epic_rows || [])
        const weeks = res.weeks || []
        const rogue = res.rogue || []
        const machE = res.machE || []
        const other = res.other || []
        const labelsRogue = res.week_labels_rogue ?? {}
        const labelsMachE = res.week_labels_mach_e ?? {}
        const labelsOther = res.week_labels_other ?? {}
        setChartData(
          weeks.map((week, i) => ({
            week,
            Rogue: rogue[i] > 0 ? Math.round(rogue[i] * 10) / 10 : null,
            MachE: machE[i] > 0 ? Math.round(machE[i] * 10) / 10 : null,
            Other: other[i] > 0 ? Math.round(other[i] * 10) / 10 : null,
            vehiclesRogue: labelsRogue[week] ?? [],
            vehiclesMachE: labelsMachE[week] ?? [],
            vehiclesOther: labelsOther[week] ?? [],
          }))
        )
      })
      .catch(() => {})
      .finally(() => setLoading(false))

    // Fetch VOS tickets
    fetch('/api/kpi/vos-tickets')
      .then((r) => r.json())
      .then((res) => {
        const weeks = res.weeks || []
        const created = res.created || []
        const resolved = res.resolved || []
        setVosData(weeks.map((week: string, i: number) => ({ week, created: created[i] || 0, resolved: resolved[i] || 0 })))
      })
      .catch(() => {})

    // Fetch Build Bugs
    fetch('/api/kpi/build-bugs')
      .then((r) => r.json())
      .then((res) => {
        const weeks = res.weeks || []
        const created = res.created || []
        const resolved = res.resolved || []
        setBugsData(weeks.map((week: string, i: number) => ({ week, created: created[i] || 0, resolved: resolved[i] || 0 })))
      })
      .catch(() => {})

    // Fetch MTBF
    fetch('/api/kpi/mtbf')
      .then((r) => r.json())
      .then((res) => {
        const weeks = res.weeks || []
        const failures = res.failures || []
        setMtbfData(weeks.map((week: string, i: number) => ({ week, failures: failures[i] || 0 })))
      })
      .catch(() => {})

    // Fetch BuildKite metrics (combined endpoint - much faster!)
    fetch('/api/kpi/buildkite-combined')
      .then((r) => r.json())
      .then((res) => {
        const weeks = res.weeks || []

        // Deployment time data
        const avgDuration = res.deployment_time?.avg_duration_mins || []
        setDeployTimeData(weeks.map((week: string, i: number) => ({
          week,
          duration: Math.round(avgDuration[i] * 10) / 10 || 0
        })))

        // Failure rate data
        const failureRate = res.failure_rate?.failure_rate || []
        const passed = res.failure_rate?.passed || []
        const failed = res.failure_rate?.failed || []
        setDeployFailureData(weeks.map((week: string, i: number) => ({
          week,
          failureRate: Math.round(failureRate[i] * 10) / 10 || 0,
          passed: passed[i] || 0,
          failed: failed[i] || 0
        })))
      })
      .catch(() => {})
  }, [])

  if (loading) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <div className="animate-spin h-10 w-10 border-2 border-primary border-t-transparent rounded-full" />
      </div>
    )
  }

  return (
    <div className="flex-1 overflow-auto">
      <div className="p-6 pl-20">
        <h1 className="text-2xl font-bold text-gray-900 mb-6">Vehicle Integration Dashboard</h1>

        {/* 3-column grid */}
        <div className="grid grid-cols-3 gap-4">
          {/* Widget 1: Cars in Build (scrollable table) */}
          <div className="bg-white rounded-lg border border-gray-300 p-4 flex flex-col h-[350px]">
            <h2 className="text-lg font-semibold text-gray-900 mb-3">Cars in Build</h2>
            <div className="overflow-auto flex-1">
              <table className="w-full text-xs border-collapse">
                <thead className="sticky top-0 bg-white">
                  <tr className="border-b border-gray-200 text-left text-gray-600">
                    <th className="py-1 pr-2">Epic</th>
                    <th className="py-1 pr-2">Start</th>
                    <th className="py-1 pr-2">Finish</th>
                    <th className="py-1 pr-2">Days</th>
                  </tr>
                </thead>
                <tbody>
                  {epicRows.map((row, idx) => (
                    <tr key={`${row.epic_key}-${idx}`} className="border-b border-gray-100 hover:bg-gray-50">
                      <td className="py-0.5 pr-2">
                        <a
                          href={`https://appliedintuition.atlassian.net/browse/${row.epic_key}`}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-primary hover:underline"
                        >
                          {row.vehicle_name || row.epic_key}
                        </a>
                      </td>
                      <td className="py-0.5 pr-2 whitespace-nowrap text-gray-700">{formatDateOnly(row.start_time)}</td>
                      <td className="py-0.5 pr-2 whitespace-nowrap text-gray-700">{formatDateOnly(row.finish_time)}</td>
                      <td className="py-0.5 pr-2 font-mono">{row.build_days}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>

          {/* Widget 2: Time in Build Chart */}
          <div className="bg-white rounded-lg border border-gray-300 p-4 h-[350px]">
            <h2 className="text-lg font-semibold text-gray-900 mb-2">Time in Build</h2>
            <p className="text-xs text-gray-500 mb-3">Average days by vehicle type</p>
            <ResponsiveContainer width="100%" height="85%">
              <LineChart data={chartData} margin={{ top: 5, right: 80, left: 0, bottom: 5 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                <XAxis dataKey="week" stroke="#6b7280" fontSize={10} />
                <YAxis stroke="#6b7280" fontSize={10} />
                <Tooltip contentStyle={{ fontSize: '12px' }} />
                <Legend wrapperStyle={{ fontSize: '11px' }} />
                <ReferenceLine y={5} stroke="#3b82f6" strokeDasharray="5 5" strokeWidth={1} label={{ value: 'Target: 5 days', fontSize: 10, fill: '#3b82f6' }} />
                <Line type="linear" dataKey="Rogue" stroke="#2563eb" strokeWidth={2} dot={{ r: 3 }} connectNulls={false}>
                  <LabelList
                    position="right"
                    content={(props: { x?: number; y?: number; width?: number; payload?: ChartPoint; index?: number }) =>
                      renderVerticalLabel(props, 'vehiclesRogue', '#2563eb', chartData)
                    }
                  />
                </Line>
                <Line type="linear" dataKey="MachE" stroke="#dc2626" strokeWidth={2} dot={{ r: 3 }} connectNulls={false}>
                  <LabelList
                    position="right"
                    content={(props: { x?: number; y?: number; width?: number; payload?: ChartPoint; index?: number }) =>
                      renderVerticalLabel(props, 'vehiclesMachE', '#dc2626', chartData)
                    }
                  />
                </Line>
                <Line type="linear" dataKey="Other" stroke="#16a34a" strokeWidth={2} dot={{ r: 3 }} strokeDasharray="4 4" connectNulls={false}>
                  <LabelList
                    position="right"
                    content={(props: { x?: number; y?: number; width?: number; payload?: ChartPoint; index?: number }) =>
                      renderVerticalLabel(props, 'vehiclesOther', '#16a34a', chartData)
                    }
                  />
                </Line>
              </LineChart>
            </ResponsiveContainer>
          </div>

          {/* Widget 3: VOS Tickets */}
          <div className="bg-white rounded-lg border border-gray-300 p-4 h-[350px]">
            <h2 className="text-lg font-semibold text-gray-900 mb-2">VOS Tickets in Build</h2>
            <p className="text-xs text-gray-500 mb-3">Vehicle OS engineer workload</p>
            <ResponsiveContainer width="100%" height="85%">
              <LineChart data={vosData} margin={{ top: 5, right: 20, left: 0, bottom: 5 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                <XAxis dataKey="week" stroke="#6b7280" fontSize={10} />
                <YAxis stroke="#6b7280" fontSize={10} />
                <Tooltip contentStyle={{ fontSize: '12px' }} />
                <Legend wrapperStyle={{ fontSize: '11px' }} />
                <ReferenceLine y={2} stroke="#3b82f6" strokeDasharray="5 5" strokeWidth={1} label={{ value: 'Target: <2', fontSize: 10, fill: '#3b82f6' }} />
                <Line type="linear" dataKey="created" stroke="#2563eb" strokeWidth={2} dot={{ r: 3 }} name="Created" />
                <Line type="linear" dataKey="resolved" stroke="#16a34a" strokeWidth={2} dot={{ r: 3 }} name="Resolved" />
              </LineChart>
            </ResponsiveContainer>
          </div>

          {/* Widget 4: Build Bugs */}
          <div className="bg-white rounded-lg border border-gray-300 p-4 h-[350px]">
            <h2 className="text-lg font-semibold text-gray-900 mb-2">Bug Tickets in Build</h2>
            <p className="text-xs text-gray-500 mb-3">Issues caught after calibration</p>
            <ResponsiveContainer width="100%" height="85%">
              <LineChart data={bugsData} margin={{ top: 5, right: 20, left: 0, bottom: 5 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                <XAxis dataKey="week" stroke="#6b7280" fontSize={10} />
                <YAxis stroke="#6b7280" fontSize={10} />
                <Tooltip contentStyle={{ fontSize: '12px' }} />
                <Legend wrapperStyle={{ fontSize: '11px' }} />
                <ReferenceLine y={1} stroke="#3b82f6" strokeDasharray="5 5" strokeWidth={1} label={{ value: 'Target: <1', fontSize: 10, fill: '#3b82f6' }} />
                <Line type="linear" dataKey="created" stroke="#dc2626" strokeWidth={2} dot={{ r: 3 }} name="Created" />
                <Line type="linear" dataKey="resolved" stroke="#16a34a" strokeWidth={2} dot={{ r: 3 }} name="Resolved" />
              </LineChart>
            </ResponsiveContainer>
          </div>

          {/* Widget 5: MTBF */}
          <div className="bg-white rounded-lg border border-gray-300 p-4 h-[350px]">
            <h2 className="text-lg font-semibold text-gray-900 mb-2">Mean Time Between Failure (MTBF)</h2>
            <p className="text-xs text-gray-500 mb-3">Vehicle stability failures (operation hours pending)</p>
            <ResponsiveContainer width="100%" height="85%">
              <LineChart data={mtbfData} margin={{ top: 5, right: 20, left: 0, bottom: 5 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                <XAxis dataKey="week" stroke="#6b7280" fontSize={10} />
                <YAxis stroke="#6b7280" fontSize={10} />
                <Tooltip contentStyle={{ fontSize: '12px' }} />
                <Legend wrapperStyle={{ fontSize: '11px' }} />
                <Line type="linear" dataKey="failures" stroke="#f59e0b" strokeWidth={2} dot={{ r: 3 }} name="Failures" />
              </LineChart>
            </ResponsiveContainer>
          </div>

          {/* Widget 6: Average Deployment Time */}
          <div className="bg-white rounded-lg border border-gray-300 p-4 h-[350px]">
            <h2 className="text-lg font-semibold text-gray-900 mb-2">Avg Deployment Time</h2>
            <p className="text-xs text-gray-500 mb-3">BuildKite deployment duration (minutes)</p>
            <ResponsiveContainer width="100%" height="85%">
              <LineChart data={deployTimeData} margin={{ top: 5, right: 20, left: 0, bottom: 5 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                <XAxis dataKey="week" stroke="#6b7280" fontSize={10} />
                <YAxis stroke="#6b7280" fontSize={10} />
                <Tooltip contentStyle={{ fontSize: '12px' }} />
                <Legend wrapperStyle={{ fontSize: '11px' }} />
                <Line type="linear" dataKey="duration" stroke="#8b5cf6" strokeWidth={2} dot={{ r: 3 }} name="Minutes" />
              </LineChart>
            </ResponsiveContainer>
          </div>

          {/* Widget 7: Deployment Failure Rate */}
          <div className="bg-white rounded-lg border border-gray-300 p-4 h-[350px]">
            <h2 className="text-lg font-semibold text-gray-900 mb-2">Deployment Failure Rate</h2>
            <p className="text-xs text-gray-500 mb-3">BuildKite deployment success vs. failure</p>
            <ResponsiveContainer width="100%" height="85%">
              <LineChart data={deployFailureData} margin={{ top: 5, right: 20, left: 0, bottom: 5 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                <XAxis dataKey="week" stroke="#6b7280" fontSize={10} />
                <YAxis stroke="#6b7280" fontSize={10} />
                <Tooltip contentStyle={{ fontSize: '12px' }} />
                <Legend wrapperStyle={{ fontSize: '11px' }} />
                <ReferenceLine y={5} stroke="#3b82f6" strokeDasharray="5 5" strokeWidth={1} label={{ value: 'Target: <5%', fontSize: 10, fill: '#3b82f6' }} />
                <Line type="linear" dataKey="failureRate" stroke="#dc2626" strokeWidth={2} dot={{ r: 3 }} name="Failure %">
                  <LabelList
                    content={(props: any) => {
                      const { x, y, index } = props
                      if (index === undefined) return null
                      const point = deployFailureData[index]
                      if (!point) return null
                      const total = (point.passed || 0) + (point.failed || 0)
                      const failed = point.failed || 0
                      return (
                        <text x={x} y={y - 15} textAnchor="middle" fontSize={8} fill="#6b7280">
                          <tspan x={x} dy={0}>{total} total</tspan>
                          <tspan x={x} dy={10}>{failed} failed</tspan>
                        </text>
                      )
                    }}
                  />
                </Line>
              </LineChart>
            </ResponsiveContainer>
          </div>
        </div>
      </div>
    </div>
  )
}

export default DashboardCompact
