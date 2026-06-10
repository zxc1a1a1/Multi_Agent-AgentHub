import { render, screen } from '@testing-library/react'
import { describe, expect, test } from 'vitest'
import ArtifactCard from './ArtifactCard'

describe('ArtifactCard', () => {
  test('renders name and mimeType', () => {
    render(
      <ArtifactCard
        id="art-1"
        name="report.pdf"
        kind="report"
        mimeType="application/pdf"
      />,
    )

    expect(screen.getByText('report.pdf')).toBeInTheDocument()
    expect(screen.getByText('application/pdf')).toBeInTheDocument()
    expect(screen.getByText('report')).toBeInTheDocument()
  })

  test('renders sourceAgent when provided', () => {
    render(
      <ArtifactCard
        name="data.csv"
        mimeType="text/csv"
        sourceAgent="code-agent"
      />,
    )

    expect(screen.getByText('data.csv')).toBeInTheDocument()
    expect(screen.getByText('text/csv')).toBeInTheDocument()
    expect(screen.getByText('by code-agent')).toBeInTheDocument()
  })

  test('renders without name using id as fallback', () => {
    render(
      <ArtifactCard
        id="fallback-id"
        kind="binary"
      />,
    )

    // When name is not provided, id is used as display text
    expect(screen.getByText('fallback-id')).toBeInTheDocument()
  })

  test('renders default label when no name or id', () => {
    render(<ArtifactCard />)

    expect(screen.getByText('Artifact')).toBeInTheDocument()
  })

  test('renders file size when provided', () => {
    render(
      <ArtifactCard
        name="large.zip"
        mimeType="application/zip"
        size={2097152}
      />,
    )

    expect(screen.getByText('large.zip')).toBeInTheDocument()
    expect(screen.getByText('2.0 MB')).toBeInTheDocument()
  })

  test('renders createdAt when provided', () => {
    const date = '2026-01-15T10:30:00Z'
    render(
      <ArtifactCard
        name="old.txt"
        createdAt={date}
      />,
    )

    expect(screen.getByText('old.txt')).toBeInTheDocument()
    // The date should be rendered (locale-dependent format)
    expect(screen.getByText(/2026/)).toBeInTheDocument()
  })

  test('renders metadata as JSON when provided', () => {
    render(
      <ArtifactCard
        name="meta.json"
        metadata={{ version: '1.0', author: 'test' }}
      />,
    )

    expect(screen.getByText('meta.json')).toBeInTheDocument()
    // Metadata is rendered as pre > JSON stringify
    const preElement = document.querySelector('pre')
    expect(preElement).toBeInTheDocument()
    expect(preElement?.textContent).toContain('"version"')
    expect(preElement?.textContent).toContain('"author"')
  })

  test('does not render metadata pre when metadata is empty', () => {
    render(
      <ArtifactCard
        name="no-meta.txt"
        metadata={{}}
      />,
    )

    expect(screen.getByText('no-meta.txt')).toBeInTheDocument()
    const preElement = document.querySelector('pre')
    expect(preElement).toBeNull()
  })
})
