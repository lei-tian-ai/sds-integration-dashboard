import { useState, useEffect } from 'react'

interface EpicRow {
  epic_key: string
  summary: string
  vehicle_name?: string
  start_time: string
  finish_time: string
  build_days: number
  week: string
  type: 'Rogue' | 'MachE' | 'Other'
}

const DetailedData = () => {
  const [activeTab, setActiveTab] = useState<string>('time-in-build')
  const [loading, setLoading] = useState(true)

  // State for all raw data
  const [timeInBuildData, setTimeInBuildData] = useState<any>(null)
  const [vosData, setVosData] = useState<any>(null)
  const [bugsData, setBugsData] = useState<any>(null)
  const [mtbfData, setMtbfData] = useState<any>(null)
  const [buildkiteData, setBuildkiteData] = useState<any>(null)

  useEffect(() => {
    setLoading(true)

    // Fetch all raw data
    Promise.all([
      fetch('/api/kpi/time-in-build?filter_id=22515').then(r => r.json()),
      fetch('/api/kpi/vos-tickets').then(r => r.json()),
      fetch('/api/kpi/build-bugs').then(r => r.json()),
      fetch('/api/kpi/mtbf').then(r => r.json()),
      fetch('/api/kpi/buildkite-combined-all').then(r => r.json()),
    ])
      .then(([timeInBuild, vos, bugs, mtbf, buildkite]) => {
        setTimeInBuildData(timeInBuild)
        setVosData(vos)
        setBugsData(bugs)
        setMtbfData(mtbf)
        setBuildkiteData(buildkite)
      })
      .catch(err => console.error('Error fetching data:', err))
      .finally(() => setLoading(false))
  }, [])

  const tabs = [
    { id: 'time-in-build', label: '#1 Time in Build' },
    { id: 'vos-tickets', label: '#2 Engineer Touch Time' },
    { id: 'mtbf', label: '#3 MTBF' },
    { id: 'build-bugs', label: '#4 Build Issues' },
    { id: 'deployment-time', label: '#5 Deployment Time' },
    { id: 'deployment-failure', label: '#6 Deployment Failure' },
  ]

  const renderJsonData = (data: any, title: string) => {
    if (!data) return <div className="text-gray-500">No data available</div>

    return (
      <div className="bg-white rounded-lg border border-gray-300 p-4">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">{title}</h3>
        <div className="overflow-auto max-h-[600px]">
          <pre className="text-xs text-gray-700 whitespace-pre-wrap">
            {JSON.stringify(data, null, 2)}
          </pre>
        </div>
      </div>
    )
  }

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
        <h1 className="text-2xl font-bold text-gray-900 mb-4">Detailed Data</h1>
        <p className="text-sm text-gray-600 mb-6">Raw API responses for each KPI widget</p>

        {/* Tabs */}
        <div className="flex border-b border-gray-300 mb-6 overflow-x-auto">
          {tabs.map(tab => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`px-4 py-2 font-medium text-sm whitespace-nowrap ${
                activeTab === tab.id
                  ? 'border-b-2 border-blue-600 text-blue-600'
                  : 'text-gray-600 hover:text-gray-900'
              }`}
            >
              {tab.label}
            </button>
          ))}
        </div>

        {/* Tab Content */}
        <div className="space-y-4">
          {activeTab === 'time-in-build' && (
            <>
              {renderJsonData(timeInBuildData, 'Time in Build - Full API Response')}
              {timeInBuildData?.epic_rows && (
                <div className="bg-white rounded-lg border border-gray-300 p-4">
                  <h3 className="text-lg font-semibold text-gray-900 mb-4">Cars Built Table</h3>
                  <div className="overflow-auto">
                    <table className="w-full text-xs border-collapse">
                      <thead className="sticky top-0 bg-gray-50">
                        <tr className="border-b border-gray-300 text-left text-gray-700 font-semibold">
                          <th className="py-2 px-3">Epic Key</th>
                          <th className="py-2 px-3">Vehicle Name</th>
                          <th className="py-2 px-3">Summary</th>
                          <th className="py-2 px-3">Type</th>
                          <th className="py-2 px-3">Start Time</th>
                          <th className="py-2 px-3">Finish Time</th>
                          <th className="py-2 px-3">Build Days</th>
                          <th className="py-2 px-3">Week</th>
                        </tr>
                      </thead>
                      <tbody>
                        {(timeInBuildData.epic_rows as EpicRow[]).map((row, idx) => (
                          <tr key={idx} className="border-b border-gray-100 hover:bg-gray-50">
                            <td className="py-1 px-3">
                              <a
                                href={`https://appliedintuition.atlassian.net/browse/${row.epic_key}`}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="text-primary hover:underline"
                              >
                                {row.epic_key}
                              </a>
                            </td>
                            <td className="py-1 px-3">{row.vehicle_name || '—'}</td>
                            <td className="py-1 px-3 max-w-md truncate">{row.summary}</td>
                            <td className="py-1 px-3">{row.type}</td>
                            <td className="py-1 px-3 whitespace-nowrap">{row.start_time}</td>
                            <td className="py-1 px-3 whitespace-nowrap">{row.finish_time}</td>
                            <td className="py-1 px-3 font-mono">{row.build_days}</td>
                            <td className="py-1 px-3">{row.week}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}
            </>
          )}

          {activeTab === 'vos-tickets' && renderJsonData(vosData, 'VOS Tickets - Full API Response')}

          {activeTab === 'mtbf' && renderJsonData(mtbfData, 'MTBF - Full API Response')}

          {activeTab === 'build-bugs' && renderJsonData(bugsData, 'Build Bugs - Full API Response')}

          {activeTab === 'deployment-time' && (
            <>
              {buildkiteData?.weekly?.deployment_time && renderJsonData(buildkiteData.weekly.deployment_time, 'Deployment Time (Weekly)')}
              {buildkiteData?.daily?.deployment_time && renderJsonData(buildkiteData.daily.deployment_time, 'Deployment Time (Daily)')}
            </>
          )}

          {activeTab === 'deployment-failure' && (
            <>
              {buildkiteData?.weekly?.failure_rate && renderJsonData(buildkiteData.weekly.failure_rate, 'Deployment Failure Rate (Weekly)')}
              {buildkiteData?.daily?.failure_rate && renderJsonData(buildkiteData.daily.failure_rate, 'Deployment Failure Rate (Daily)')}
            </>
          )}
        </div>
      </div>
    </div>
  )
}

export default DetailedData
