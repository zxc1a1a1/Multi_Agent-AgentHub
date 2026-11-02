import { useEffect, useState, type MouseEvent } from 'react'
import { Box } from 'lucide-react'
import * as api from '../../services/api'

interface Props {
  id?: string
  name?: string
  kind?: string
  mimeType?: string
  size?: number | string
  downloadPath?: string
  createdAt?: string
  sourceAgent?: string
  metadata?: Record<string, unknown>
}

function formatSize(size?: number | string): string {
  if (size === undefined || size === '') return ''
  const n = typeof size === 'string' ? parseInt(size, 10) : size
  if (!Number.isFinite(n) || n <= 0) return ''
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / (1024 * 1024)).toFixed(1)} MB`
}

export default function ArtifactCard({ id, name, kind, mimeType, size, downloadPath, createdAt, sourceAgent, metadata }: Props) {
  const contentURL = id ? `/api/artifacts/${encodeURIComponent(id)}/content` : ''
  const isImage = Boolean(contentURL && mimeType?.toLowerCase().startsWith('image/'))
  const [imageURL, setImageURL] = useState('')

  useEffect(() => {
    if (!isImage || !id || typeof URL.createObjectURL !== 'function') return
    let active = true
    let objectURL = ''
    api.fetchArtifactContent(id)
      .then((blob) => {
        objectURL = URL.createObjectURL(blob)
        if (active) setImageURL(objectURL)
      })
      .catch(() => {})
    return () => {
      active = false
      if (objectURL) URL.revokeObjectURL(objectURL)
    }
  }, [id, isImage])

  const openArtifact = async (event: MouseEvent) => {
    if (!id || typeof URL.createObjectURL !== 'function') return
    event.preventDefault()
    try {
      const blob = await api.fetchArtifactContent(id)
      const objectURL = URL.createObjectURL(blob)
      window.open(objectURL, '_blank', 'noopener')
      setTimeout(() => URL.revokeObjectURL(objectURL), 60_000)
    } catch {
      window.open(contentURL, '_blank', 'noopener')
    }
  }

  return (
    <div className="mt-2 rounded-lg border border-gray-200 bg-white p-3">
      <div className="flex items-start gap-3">
        <div className="w-9 h-9 rounded-lg bg-violet-50 flex items-center justify-center flex-shrink-0">
          <Box className="w-4 h-4 text-violet-500" />
        </div>
        <div className="min-w-0 flex-1">
          <p className="text-sm font-medium text-gray-800 truncate">{name || id || 'Artifact'}</p>
          <div className="flex flex-wrap gap-2 text-xs text-gray-400 mt-0.5">
            {kind && <span>{kind}</span>}
            {mimeType && <span>{mimeType}</span>}
            {formatSize(size) && <span>{formatSize(size)}</span>}
            {sourceAgent && <span>by {sourceAgent}</span>}
            {createdAt && <span>{new Date(createdAt).toLocaleString()}</span>}
          </div>
          {contentURL && (
            <a
              href={contentURL}
              target="_blank"
              rel="noreferrer"
              onClick={openArtifact}
              className="inline-block mt-2 text-xs text-indigo-600 hover:text-indigo-700 underline"
            >
              Open artifact
            </a>
          )}
          {isImage && (
            <img
              src={imageURL || contentURL}
              alt={name || 'artifact preview'}
              loading="lazy"
              className="mt-2 max-h-80 w-auto max-w-full rounded border border-gray-200 object-contain"
            />
          )}
          {downloadPath && <p className="text-[11px] text-gray-400 mt-1">Download metadata path: {downloadPath}</p>}
          {metadata && Object.keys(metadata).length > 0 && (
            <pre className="mt-2 text-[11px] text-gray-500 bg-gray-50 border border-gray-100 rounded p-2 max-h-24 overflow-auto">{JSON.stringify(metadata, null, 2)}</pre>
          )}
        </div>
      </div>
    </div>
  )
}
