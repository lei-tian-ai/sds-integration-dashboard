interface SidebarProps {
  activePage: string
  setActivePage: (id: string) => void
  isOpen: boolean
  setIsOpen: (open: boolean) => void
}

const Sidebar = ({ activePage, setActivePage, isOpen, setIsOpen }: SidebarProps) => {
  const pages = [
    { id: 'dashboard', label: 'KPI Dashboard' },
    { id: 'detailed-data', label: 'Detailed Data' },
    { id: 'about', label: 'About' },
  ]

  return (
    <>
      {/* Backdrop */}
      {isOpen && (
        <div
          className="fixed inset-0 bg-black bg-opacity-50 z-40"
          onClick={() => setIsOpen(false)}
        />
      )}

      {/* Sidebar */}
      <div className={`fixed left-0 top-0 h-full w-64 bg-white border-r border-gray-300 flex flex-col z-50 transition-transform duration-300 ${
        isOpen ? 'translate-x-0' : '-translate-x-full'
      }`}>
        <div className="p-6 border-b border-gray-300">
          <div className="flex items-center gap-3">
            <img
              src="/applied-logo.png"
              alt="Applied Intuition"
              className="h-10 w-auto"
            />
            <div className="text-xl font-bold text-gray-900">
              Vehicle Integration
            </div>
          </div>
        </div>

        <nav className="flex-1 p-4">
          <ul className="space-y-1">
            {pages.map((page) => (
              <li key={page.id}>
                <button
                  onClick={() => {
                    setActivePage(page.id)
                    setIsOpen(false)
                  }}
                  className={`w-full text-left px-4 py-2 rounded-lg transition-colors ${
                    activePage === page.id
                      ? 'bg-primary text-white'
                      : 'text-gray-700 hover:bg-gray-100'
                  }`}
                >
                  {page.label}
                </button>
              </li>
            ))}
          </ul>
        </nav>

        <div className="p-4 border-t border-gray-300">
          <div className="text-xs text-gray-500">Version 1.0.0</div>
        </div>
      </div>
    </>
  )
}

export default Sidebar
