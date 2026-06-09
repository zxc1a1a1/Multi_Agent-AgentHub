interface Props {
  enabled: boolean
  healthy: boolean
}

const baseClass = 'inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium'

export default function AgentStatusBadge({ enabled, healthy }: Props) {
  if (!enabled) {
    return (
      <span className={`${baseClass} bg-gray-100 text-gray-500`}>
        <span className="w-1.5 h-1.5 rounded-full bg-gray-400" />
        Disabled
      </span>
    )
  }
  if (healthy) {
    return (
      <span className={`${baseClass} bg-green-50 text-green-700`}>
        <span className="w-1.5 h-1.5 rounded-full bg-green-500" />
        Healthy
      </span>
    )
  }
  return (
    <span className={`${baseClass} bg-red-50 text-red-600`}>
      <span className="w-1.5 h-1.5 rounded-full bg-red-400" />
      Unhealthy
    </span>
  )
}
