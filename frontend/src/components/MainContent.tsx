import { useState, useEffect } from 'react'
import DashboardCompact from './DashboardCompact'
import About from './About'

interface ApiResponse {
  message: string
}

interface MainContentProps {
  activePage: string
}

const MainContent = ({ activePage }: MainContentProps) => {
  const [message, setMessage] = useState<string>('Loading...')
  const [error, setError] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    fetch('/api/hello')
      .then(response => response.json())
      .then((data: ApiResponse) => {
        setMessage(data.message)
        setIsLoading(false)
      })
      .catch(err => {
        setError(err.message)
        setIsLoading(false)
      })
  }, [])

  if (activePage === 'dashboard') {
    return <DashboardCompact />
  }

  if (activePage === 'about') {
    return <About />
  }

  return (
    <div className="flex-1 overflow-auto">
      <div className="p-8">
        <div className="max-w-4xl">
          <h1 className="text-4xl font-bold text-gray-900 mb-6">
            Hello Applied Intuition
          </h1>
          
          <div className="bg-white rounded-lg border border-gray-300 p-6 mb-6">
            <h2 className="text-xl font-semibold text-gray-900 mb-4">
              API Response
            </h2>
            
            {isLoading ? (
              <div className="flex items-center text-gray-700">
                <div className="animate-spin mr-3 h-5 w-5 border-2 border-primary border-t-transparent rounded-full"></div>
                Loading...
              </div>
            ) : error ? (
              <div className="text-red p-4 bg-red bg-opacity-10 rounded-lg">
                Error: {error}
              </div>
            ) : (
              <div className="text-primary text-lg font-medium">
                {message}
              </div>
            )}
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="bg-white rounded-lg border border-gray-300 p-5">
              <div className="text-sm text-gray-500 mb-1">Backend</div>
              <div className="text-lg font-semibold text-gray-900">Go + Gin</div>
            </div>
            
            <div className="bg-white rounded-lg border border-gray-300 p-5">
              <div className="text-sm text-gray-500 mb-1">Frontend</div>
              <div className="text-lg font-semibold text-gray-900">React + TypeScript</div>
            </div>
            
            <div className="bg-white rounded-lg border border-gray-300 p-5">
              <div className="text-sm text-gray-500 mb-1">Styling</div>
              <div className="text-lg font-semibold text-gray-900">Tailwind CSS</div>
            </div>
            
            <div className="bg-white rounded-lg border border-gray-300 p-5">
              <div className="text-sm text-gray-500 mb-1">Deployment</div>
              <div className="text-lg font-semibold text-gray-900">Apps Platform</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

export default MainContent
