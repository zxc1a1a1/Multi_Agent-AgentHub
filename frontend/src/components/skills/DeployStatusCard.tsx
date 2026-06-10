interface Props {
  environment?: string
  version?: string
  status?: string
  timestamp?: string
}

const statusColors: Record<string, string> = {
  success: 'bg-green-100 text-green-700 border-green-300',
  deployed: 'bg-green-100 text-green-700 border-green-300',
  running: 'bg-blue-100 text-blue-700 border-blue-300',
  failed: 'bg-red-100 text-red-700 border-red-300',
  pending: 'bg-amber-100 text-amber-700 border-amber-300',
  rollback: 'bg-orange-100 text-orange-700 border-orange-300',
}

export default function DeployStatusCard({
  environment,
  version,
  status,
  timestamp,
}: Props) {
  const statusKey = (status || 'pending').toLowerCase()
  const colorClass =
    statusColors[statusKey] ||
    'bg-gray-100 text-gray-700 border-gray-300'

  return (
    <div className="mt-2 rounded-lg border border-gray-200 bg-white p-4">
      <div className="flex items-center justify-between mb-2">
        <h4 className="text-sm font-semibold text-gray-800">Deployment Status</h4>
        <span
          className={`text-xs px-2 py-0.5 rounded-full border ${colorClass}`}
        >
          {status || 'pending'}
        </span>
      </div>

      <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs">
        {environment && (
          <>
            <span className="text-gray-400">Environment</span>
            <span className="text-gray-700 font-mono">{environment}</span>
          </>
        )}
        {version && (
          <>
            <span className="text-gray-400">Version</span>
            <span className="text-gray-700 font-mono">{version}</span>
          </>
        )}
        {timestamp && (
          <>
            <span className="text-gray-400">Timestamp</span>
            <span className="text-gray-700">
              {new Date(timestamp).toLocaleString()}
            </span>
          </>
        )}
      </div>
    </div>
  )
}
