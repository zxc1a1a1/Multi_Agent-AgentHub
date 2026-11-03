import { useMemo } from 'react'

/**
 * R3-05: declarative chart. Input is plain JSON (type/data/series); no
 * functions, expressions or eval. Rendering is pure SVG so the component is
 * small and can be lazy-loaded. Recharts was evaluated but not added: the
 * required bar/line/area/pie shapes do not justify a ~500KB dependency.
 */

export const CHART_MAX_POINTS = 5000
export const CHART_MAX_SERIES = 10
export const CHART_MAX_PAYLOAD_BYTES = 256 * 1024

const CHART_TYPES = ['line', 'bar', 'area', 'pie'] as const
type ChartType = (typeof CHART_TYPES)[number]

type Series = { key: string; name?: string; color?: string }
type Point = { label?: string; values: number[] }
type ChartSpec = { type: ChartType; title?: string; series: Series[]; data: Point[] }

export type ChartValidation = { ok: true; spec: ChartSpec } | { ok: false; reason: string }

const DEFAULT_COLORS = ['#6366f1', '#10b981', '#f59e0b', '#f43f5e', '#06b6d4', '#8b5cf6', '#84cc16', '#ec4899', '#14b8a6', '#f97316']

function isPlainObject(value: unknown): value is Record<string, unknown> {
  if (value === null || typeof value !== 'object' || Array.isArray(value)) return false
  const proto = Object.getPrototypeOf(value)
  return proto === Object.prototype || proto === null
}

function isFiniteNumber(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value)
}

function cleanText(value: unknown, limit: number): string {
  return typeof value === 'string' ? value.slice(0, limit) : ''
}

export function validateChartProps(props: unknown): ChartValidation {
  if (!isPlainObject(props)) return { ok: false, reason: 'props must be an object' }
  let encoded: string
  try {
    encoded = JSON.stringify(props)
  } catch {
    return { ok: false, reason: 'props are not JSON' }
  }
  if (encoded.length > CHART_MAX_PAYLOAD_BYTES) return { ok: false, reason: 'payload exceeds 256KB' }
  const type = props.type
  if (typeof type !== 'string' || !(CHART_TYPES as readonly string[]).includes(type)) {
    return { ok: false, reason: 'unsupported chart type' }
  }
  const rawSeries = props.series
  if (!Array.isArray(rawSeries) || rawSeries.length === 0 || rawSeries.length > CHART_MAX_SERIES) {
    return { ok: false, reason: 'series must be 1..10' }
  }
  const series: Series[] = []
  for (const item of rawSeries) {
    if (!isPlainObject(item)) return { ok: false, reason: 'series entries must be objects' }
    const key = cleanText(item.key, 40)
    if (!key) return { ok: false, reason: 'series key is required' }
    series.push({
      key,
      name: cleanText(item.name, 80) || undefined,
      color: /^#[0-9a-f]{6}$/i.test(cleanText(item.color, 7)) ? (item.color as string) : undefined,
    })
  }
  const rawData = props.data
  if (!Array.isArray(rawData) || rawData.length === 0 || rawData.length > CHART_MAX_POINTS) {
    return { ok: false, reason: 'data must be 1..5000 points' }
  }
  const data: Point[] = []
  for (const item of rawData) {
    if (!isPlainObject(item)) return { ok: false, reason: 'data entries must be objects' }
    const values = Array.isArray(item.values) ? item.values : []
    if (values.length !== series.length || !values.every(isFiniteNumber)) {
      return { ok: false, reason: 'each point needs one finite value per series' }
    }
    data.push({ label: cleanText(item.label, 80), values: values as number[] })
  }
  return { ok: true, spec: { type: type as ChartType, title: cleanText(props.title, 120), series, data } }
}

function scale(values: number[]): { min: number; max: number } {
  let min = 0
  let max = 1
  for (const value of values) {
    if (value < min) min = value
    if (value > max) max = value
  }
  return { min, max: max === min ? min + 1 : max }
}

function LineSeries({ spec }: { spec: ChartSpec }) {
  const width = 640
  const height = 220
  const pad = 24
  const flat = spec.data.flatMap((point) => point.values)
  const { min, max } = scale(flat)
  const stepX = spec.data.length > 1 ? (width - pad * 2) / (spec.data.length - 1) : 0
  const y = (value: number) => height - pad - ((value - min) / (max - min)) * (height - pad * 2)

  return (
    <svg viewBox={`0 0 ${width} ${height}`} className="w-full" role="img" aria-label="line chart">
      <line x1={pad} y1={height - pad} x2={width - pad} y2={height - pad} stroke="#e5e7eb" />
      {spec.series.map((series, index) => {
        const color = series.color || DEFAULT_COLORS[index % DEFAULT_COLORS.length]
        const points = spec.data.map((point, i) => `${pad + i * stepX},${y(point.values[index])}`).join(' ')
        const area = `${pad},${height - pad} ${points} ${pad + (spec.data.length - 1) * stepX},${height - pad}`
        return (
          <g key={series.key}>
            {spec.type === 'area' && <polygon points={area} fill={color} opacity={0.15} />}
            <polyline points={points} fill="none" stroke={color} strokeWidth={2} />
            {spec.data.map((point, i) => (
              <circle key={i} cx={pad + i * stepX} cy={y(point.values[index])} r={2.5} fill={color} />
            ))}
          </g>
        )
      })}
    </svg>
  )
}

function BarSeries({ spec }: { spec: ChartSpec }) {
  const width = 640
  const height = 220
  const pad = 24
  const flat = spec.data.flatMap((point) => point.values)
  const { min, max } = scale(flat)
  const groupWidth = (width - pad * 2) / spec.data.length
  const barWidth = Math.max(groupWidth / (spec.series.length + 1), 2)
  return (
    <svg viewBox={`0 0 ${width} ${height}`} className="w-full" role="img" aria-label="bar chart">
      <line x1={pad} y1={height - pad} x2={width - pad} y2={height - pad} stroke="#e5e7eb" />
      {spec.data.map((point, i) =>
        spec.series.map((series, s) => {
          const value = point.values[s]
          const h = ((value - min) / (max - min)) * (height - pad * 2)
          return (
            <rect
              key={`${i}-${series.key}`}
              x={pad + i * groupWidth + s * barWidth + 1}
              y={height - pad - h}
              width={Math.max(barWidth - 2, 1)}
              height={Math.max(h, 1)}
              fill={series.color || DEFAULT_COLORS[s % DEFAULT_COLORS.length]}
            />
          )
        }),
      )}
    </svg>
  )
}

function PieSeries({ spec }: { spec: ChartSpec }) {
  const values = spec.data.map((point) => point.values[0] ?? 0)
  const total = values.reduce((sum, value) => sum + Math.max(value, 0), 0)
  let angle = -Math.PI / 2
  const radius = 80
  const center = 100

  return (
    <svg viewBox="0 0 200 200" className="mx-auto h-48" role="img" aria-label="pie chart">
      {spec.data.map((point, i) => {
        const value = Math.max(point.values[0] ?? 0, 0)
        const sweep = total > 0 ? (value / total) * Math.PI * 2 : 0
        const x1 = center + radius * Math.cos(angle)
        const y1 = center + radius * Math.sin(angle)
        angle += sweep
        const x2 = center + radius * Math.cos(angle)
        const y2 = center + radius * Math.sin(angle)
        const large = sweep > Math.PI ? 1 : 0
        const path = `M ${center} ${center} L ${x1} ${y1} A ${radius} ${radius} 0 ${large} 1 ${x2} ${y2} Z`
        return <path key={i} d={path} fill={spec.series[0]?.color || DEFAULT_COLORS[i % DEFAULT_COLORS.length]} />
      })}
    </svg>
  )
}

export default function DeclarativeChart({ props }: { props: unknown }) {
  const result = useMemo(() => validateChartProps(props), [props])
  if (!result.ok) {
    return (
      <div className="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800" role="alert">
        <div className="font-medium">Chart rejected</div>
        <div className="mt-1 text-xs">reason: {result.reason}</div>
      </div>
    )
  }
  const spec = result.spec
  return (
    <div className="mt-2 rounded-lg border border-gray-200 bg-white p-4">
      {spec.title && <div className="mb-2 text-sm font-medium text-gray-700">{spec.title}</div>}
      {spec.type === 'pie' ? <PieSeries spec={spec} /> : spec.type === 'bar' ? <BarSeries spec={spec} /> : <LineSeries spec={spec} />}
      <div className="mt-2 flex flex-wrap gap-3">
        {spec.series.map((series, index) => (
          <span key={series.key} className="inline-flex items-center gap-1 text-xs text-gray-600">
            <span
              className="h-2.5 w-2.5 rounded-full"
              style={{ backgroundColor: series.color || DEFAULT_COLORS[index % DEFAULT_COLORS.length] }}
            />
            {series.name || series.key}
          </span>
        ))}
      </div>
    </div>
  )
}
