import { useState } from 'react'
import { Image, AlertTriangle } from 'lucide-react'

interface Props {
  url?: string
  alt?: string
}

/**
 * Image preview from a safe URL.
 * NO upload, NO file storage.
 */
export default function ImagePreview({ url, alt }: Props) {
  const [error, setError] = useState(false)

  if (!url || error) {
    return (
      <div className="mt-2 rounded-lg border border-gray-200 bg-white p-4">
        <div className="flex flex-col items-center gap-2 text-gray-400">
          {error ? (
            <>
              <AlertTriangle className="w-8 h-8 text-amber-400" />
              <span className="text-xs">Failed to load image</span>
            </>
          ) : (
            <>
              <Image className="w-8 h-8" />
              <span className="text-xs">No image URL provided</span>
            </>
          )}
        </div>
      </div>
    )
  }

  return (
    <div className="mt-2 rounded-lg border border-gray-200 bg-white overflow-hidden">
      <img
        src={url}
        alt={alt || 'Preview'}
        onError={() => setError(true)}
        className="w-full max-h-96 object-contain bg-gray-50"
        loading="lazy"
      />
      {alt && (
        <div className="px-3 py-1.5 bg-gray-50 border-t border-gray-100">
          <p className="text-xs text-gray-500 truncate">{alt}</p>
        </div>
      )}
    </div>
  )
}
