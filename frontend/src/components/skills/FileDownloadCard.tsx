import { File, Download } from 'lucide-react'

interface Props {
  filename?: string
  size?: number | string
  mimeType?: string
  createdAt?: string
}

function formatFileSize(bytes: number | string | undefined): string {
  if (bytes === undefined || bytes === '') return ''
  const n = typeof bytes === 'string' ? parseInt(bytes, 10) : bytes
  if (isNaN(n) || n <= 0) return ''
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / (1024 * 1024)).toFixed(1)} MB`
}

/**
 * File metadata card — METADATA ONLY, no actual download.
 */
export default function FileDownloadCard({
  filename,
  size,
  mimeType,
  createdAt,
}: Props) {
  return (
    <div className="mt-2 rounded-lg border border-gray-200 bg-white p-3 flex items-center gap-3">
      <div className="w-10 h-10 rounded-lg bg-blue-50 flex items-center justify-center flex-shrink-0">
        <File className="w-5 h-5 text-blue-500" />
      </div>

      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium text-gray-800 truncate">
          {filename || 'Unnamed file'}
        </p>
        <div className="flex items-center gap-2 text-xs text-gray-400 mt-0.5">
          {mimeType && <span>{mimeType}</span>}
          {formatFileSize(size) && <span>{formatFileSize(size)}</span>}
          {createdAt && (
            <span>{new Date(createdAt).toLocaleDateString()}</span>
          )}
        </div>
      </div>

      <div className="flex-shrink-0 text-gray-300" title="Metadata only — download not available">
        <Download className="w-4 h-4" />
      </div>
    </div>
  )
}
