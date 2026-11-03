import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import DeclarativeChart, {
  CHART_MAX_POINTS,
  CHART_MAX_SERIES,
  validateChartProps,
} from './DeclarativeChart'

function spec(overrides: Record<string, unknown> = {}) {
  return {
    type: 'line',
    series: [{ key: 'a' }],
    data: [{ label: 'one', values: [1] }, { label: 'two', values: [3] }],
    ...overrides,
  }
}

describe('validateChartProps', () => {
  it('accepts line/bar/area/pie', () => {
    for (const type of ['line', 'bar', 'area', 'pie']) {
      expect(validateChartProps(spec({ type })).ok).toBe(true)
    }
  })

  it('rejects unsupported types and malformed series/data', () => {
    expect(validateChartProps(spec({ type: 'radar' })).ok).toBe(false)
    expect(validateChartProps(spec({ series: [] })).ok).toBe(false)
    expect(validateChartProps(spec({ data: [{ label: 'x', values: [1, 2] }] })).ok).toBe(false)
    expect(validateChartProps(spec({ data: [{ label: 'x', values: [Number.NaN] }] })).ok).toBe(false)
  })

  it('enforces series, point and payload limits', () => {
    const manySeries = Array.from({ length: CHART_MAX_SERIES + 1 }, (_, i) => ({ key: `s${i}` }))
    expect(validateChartProps(spec({ series: manySeries })).ok).toBe(false)
    const manyPoints = Array.from({ length: CHART_MAX_POINTS + 1 }, () => ({ values: [1] }))
    expect(validateChartProps(spec({ data: manyPoints })).ok).toBe(false)
    const huge = spec({ title: 'x'.repeat(300 * 1024) })
    expect(validateChartProps(huge).ok).toBe(false)
  })

  it('ignores unknown fields instead of executing them', () => {
    const result = validateChartProps(spec({ data: [{ label: 'x', values: [1], render: 'javascript:1' }] }))
    expect(result.ok).toBe(true)
  })
})

describe('DeclarativeChart', () => {
  it('renders a line chart with a legend', () => {
    const { container } = render(<DeclarativeChart props={spec({ title: 'Trend', series: [{ key: 'a', name: 'Alpha' }] })} />)
    expect(screen.getByText('Trend')).toBeInTheDocument()
    expect(screen.getByText('Alpha')).toBeInTheDocument()
    expect(container.querySelector('svg[aria-label="line chart"]')).not.toBeNull()
  })

  it('renders bar, area and pie variants', () => {
    for (const [type, label] of [
      ['bar', 'bar chart'],
      ['area', 'line chart'],
      ['pie', 'pie chart'],
    ] as const) {
      const { container, unmount } = render(<DeclarativeChart props={spec({ type })} />)
      expect(container.querySelector(`svg[aria-label="${label}"]`)).not.toBeNull()
      unmount()
    }
  })

  it('shows a reason instead of rendering an invalid spec', () => {
    const { container } = render(<DeclarativeChart props={{ type: 'radar' }} />)
    expect(screen.getByRole('alert')).toHaveTextContent('Chart rejected')
    expect(container.querySelector('svg')).toBeNull()
  })

  it('escapes labels as text', () => {
    const { container } = render(
      <DeclarativeChart props={spec({ data: [{ label: '<img src=x onerror=alert(1)>', values: [1] }] })} />,
    )
    expect(container.querySelector('img')).toBeNull()
    expect(container.innerHTML).not.toContain('onerror')
  })
})
