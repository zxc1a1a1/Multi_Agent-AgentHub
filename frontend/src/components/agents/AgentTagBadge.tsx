interface Props {
  label: string
  variant?: 'default' | 'source'
}

export default function AgentTagBadge({ label, variant = 'default' }: Props) {
  if (variant === 'source') {
    const isDynamic = label === 'dynamic'
    return (
      <span
        className={`inline-block px-1.5 py-0.5 rounded text-xs font-medium ${
          isDynamic
            ? 'bg-purple-50 text-purple-600 border border-purple-200'
            : 'bg-blue-50 text-blue-600 border border-blue-200'
        }`}
      >
        {label}
      </span>
    )
  }

  return (
    <span className="inline-block px-1.5 py-0.5 rounded bg-gray-100 text-gray-600 text-xs font-medium">
      {label}
    </span>
  )
}
