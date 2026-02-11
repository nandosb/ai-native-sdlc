import { useState, useEffect } from 'react'
import { X, FileText, Loader2 } from 'lucide-react'
import { fetchArtifact } from '../lib/api'

interface Props {
  artifactKey: string
  onClose: () => void
}

export function ArtifactViewer({ artifactKey, onClose }: Props) {
  const [content, setContent] = useState<string | null>(null)
  const [path, setPath] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    setLoading(true)
    setError(null)
    fetchArtifact(artifactKey)
      .then((res) => {
        setContent(res.content)
        setPath(res.path)
      })
      .catch((err) => {
        setError(err instanceof Error ? err.message : 'Failed to load artifact')
      })
      .finally(() => setLoading(false))
  }, [artifactKey])

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose} />
      <div className="relative bg-gray-900 border border-gray-800 rounded-xl shadow-2xl w-full max-w-3xl mx-4 max-h-[80vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-gray-800 shrink-0">
          <div className="flex items-center gap-3">
            <FileText size={18} className="text-blue-400" />
            <div>
              <h2 className="text-lg font-semibold text-white capitalize">{artifactKey.replace(/_/g, ' ')}</h2>
              {path && <p className="text-xs text-gray-500 font-mono mt-0.5">{path}</p>}
            </div>
          </div>
          <button onClick={onClose} className="text-gray-500 hover:text-gray-300">
            <X size={20} />
          </button>
        </div>

        {/* Content */}
        <div className="px-6 py-4 overflow-y-auto flex-1 min-h-0">
          {loading && (
            <div className="flex items-center justify-center py-12">
              <Loader2 size={24} className="text-blue-400 animate-spin" />
            </div>
          )}
          {error && (
            <div className="text-sm text-red-400 py-4">{error}</div>
          )}
          {content !== null && !loading && (
            <pre className="text-sm text-gray-300 whitespace-pre-wrap font-mono leading-relaxed">
              {content}
            </pre>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end px-6 py-3 border-t border-gray-800 shrink-0">
          <button
            onClick={onClose}
            className="px-4 py-2 rounded-lg text-sm font-medium text-gray-400 hover:text-gray-200 transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  )
}
