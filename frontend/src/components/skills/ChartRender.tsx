import { useMemo } from 'react'

interface ChartDataPoint {
  label?: string
  value: number
}

interface Props {
  chartType?: 'bar' | 'line' | 'pie'
  data?: ChartDataPoint[]
}

/**
 * Lightweight chart renderer using pure CSS/SVG.
 * No interactive editing.
 */
export default function ChartRender({ chartType = 'bar', data = [] }: Props) {
  const maxValue = useMemo(() => {
    if (data.length === 0) return 1
    return Math.max(...data.map((d) => d.value), 1)
  }, [data])

  const barColors = [
    'bg-indigo-500',
    'bg-emerald-500',
    'bg-amber-500',
    'bg-rose-500',
    'bg-cyan-500',
    'bg-violet-500',
  ]

  if (data.length === 0) {
    return (
      <div className="mt-2 rounded-lg border border-gray-200 bg-white p-4 text-center text-xs text-gray-400">
        No chart data
      </div>
    )
  }

  return (
    <div className="mt-2 rounded-lg border border-gray-200 bg-white p-4">
      <div className="flex items-center gap-2 mb-3">
        <span className="text-xs text-gray-400 uppercase tracking-wide">
          {chartType} chart
        </span>
        <span className="text-[10px] text-gray-300">
          {data.length} data point{data.length !== 1 ? 's' : ''}
        </span>
      </div>

      {chartType === 'bar' && (
        <div className="space-y-2">
          {data.map((point, i) => (
            <div key={i} className="flex items-center gap-2">
              <span className="text-xs text-gray-500 w-20 truncate flex-shrink-0">
                {point.label || `#${i + 1}`}
              </span>
              <div className="flex-1 bg-gray-100 rounded-full h-4 overflow-hidden">
                <div
                  className={`h-full rounded-full ${barColors[i % barColors.length]} transition-all`}
                  style={{
                    width: `${Math.max((point.value / maxValue) * 100, 2)}%`,
                  }}
                />
              </div>
              <span className="text-xs text-gray-600 w-12 text-right flex-shrink-0">
                {point.value}
              </span>
            </div>
          ))}
        </div>
      )}

      {chartType === 'pie' && (
        <div className="flex flex-wrap gap-2">
          {data.map((point, i) => {
            const total = data.reduce((sum, d) => sum + d.value, 0)
            const pct = total > 0 ? ((point.value / total) * 100).toFixed(1) : '0'
            return (
              <div
                key={i}
                className="inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-full border border-gray-200"
              >
                <span
                  className={`w-2.5 h-2.5 rounded-full ${barColors[i % barColors.length]}`}
                />
                <span className="text-xs text-gray-600">
                  {point.label || `Item ${i + 1}`}: {pct}%
                </span>
              </div>
            )
          })}
        </div>
      )}

      {chartType === 'line' && (
        <div className="h-24 flex items-end gap-1">
          {data.map((point, i) => {
            const height = Math.max((point.value / maxValue) * 100, 4)
            return (
              <div
                key={i}
                className="flex-1 flex flex-col items-center gap-1"
              >
                <span className="text-[10px] text-gray-400">{point.value}</span>
                <div
                  className={`w-full rounded-t ${barColors[i % barColors.length]}`}
                  style={{ height: `${height}%` }}
                />
                <span className="text-[10px] text-gray-400 truncate w-full text-center">
                  {point.label || ''}
                </span>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
