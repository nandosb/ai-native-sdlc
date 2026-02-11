import { useState, useEffect } from 'react'
import { fetchManifest, saveManifest, type RepoConfig } from '../lib/api'
import { Plus, Trash2, Save } from 'lucide-react'

const LANGUAGES = ['auto', 'go', 'typescript', 'python'] as const

const emptyRepo = (): RepoConfig => ({ name: '', path: '', team: '' })

export function ManifestEditor() {
  const [prd, setPrd] = useState('')
  const [repos, setRepos] = useState<RepoConfig[]>([emptyRepo()])
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchManifest()
      .then((data) => {
        if (data.prd) setPrd(data.prd)
        if (data.repos?.length) setRepos(data.repos)
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  const updateRepo = (index: number, field: keyof RepoConfig, value: string) => {
    setRepos((prev) => prev.map((r, i) => (i === index ? { ...r, [field]: value } : r)))
  }

  const addRepo = () => setRepos((prev) => [...prev, emptyRepo()])

  const removeRepo = (index: number) => {
    setRepos((prev) => prev.filter((_, i) => i !== index))
  }

  const handleSave = async () => {
    setMessage(null)
    setSaving(true)
    try {
      await saveManifest({ prd, repos })
      setMessage({ type: 'success', text: 'Manifest saved. State reloaded.' })
    } catch (err: unknown) {
      const text = err instanceof Error ? err.message : 'Save failed'
      setMessage({ type: 'error', text })
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return <div className="text-gray-400 text-sm py-8 text-center">Loading manifest...</div>
  }

  return (
    <div className="space-y-6">
      <div className="card">
        <h2 className="text-lg font-semibold text-white mb-4">Manifest Configuration</h2>

        {/* PRD URL */}
        <div className="mb-6">
          <label className="block text-sm font-medium text-gray-300 mb-1">PRD URL</label>
          <input
            type="text"
            value={prd}
            onChange={(e) => setPrd(e.target.value)}
            placeholder="https://notion.so/your-prd or path/to/prd.md"
            className="w-full bg-gray-800 border border-gray-700 rounded-md px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>

        {/* Repos table */}
        <div className="mb-4">
          <label className="block text-sm font-medium text-gray-300 mb-2">Repositories</label>
          <div className="space-y-2">
            {/* Header */}
            <div className="grid grid-cols-[1fr_1fr_1fr_120px_40px] gap-2 text-xs text-gray-500 px-1">
              <span>Name</span>
              <span>Path</span>
              <span>Team</span>
              <span>Language</span>
              <span></span>
            </div>

            {repos.map((repo, i) => (
              <div key={i} className="grid grid-cols-[1fr_1fr_1fr_120px_40px] gap-2">
                <input
                  type="text"
                  value={repo.name}
                  onChange={(e) => updateRepo(i, 'name', e.target.value)}
                  placeholder="repo-name"
                  className="bg-gray-800 border border-gray-700 rounded-md px-2 py-1.5 text-sm text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                <input
                  type="text"
                  value={repo.path}
                  onChange={(e) => updateRepo(i, 'path', e.target.value)}
                  placeholder="/path/to/repo"
                  className="bg-gray-800 border border-gray-700 rounded-md px-2 py-1.5 text-sm text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                <input
                  type="text"
                  value={repo.team}
                  onChange={(e) => updateRepo(i, 'team', e.target.value)}
                  placeholder="team-name"
                  className="bg-gray-800 border border-gray-700 rounded-md px-2 py-1.5 text-sm text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                <select
                  value={repo.language || 'auto'}
                  onChange={(e) =>
                    updateRepo(i, 'language', e.target.value === 'auto' ? '' : e.target.value)
                  }
                  className="bg-gray-800 border border-gray-700 rounded-md px-2 py-1.5 text-sm text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  {LANGUAGES.map((lang) => (
                    <option key={lang} value={lang}>
                      {lang}
                    </option>
                  ))}
                </select>
                <button
                  onClick={() => removeRepo(i)}
                  disabled={repos.length <= 1}
                  className="flex items-center justify-center text-gray-500 hover:text-red-400 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
                >
                  <Trash2 size={16} />
                </button>
              </div>
            ))}
          </div>

          <button
            onClick={addRepo}
            className="mt-2 flex items-center gap-1.5 text-sm text-blue-400 hover:text-blue-300 transition-colors"
          >
            <Plus size={14} />
            Add repo
          </button>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-3 pt-4 border-t border-gray-800">
          <button
            onClick={handleSave}
            disabled={saving}
            className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white text-sm font-medium rounded-md transition-colors"
          >
            <Save size={16} />
            {saving ? 'Saving...' : 'Save'}
          </button>

          {message && (
            <span
              className={`text-sm ${message.type === 'success' ? 'text-green-400' : 'text-red-400'}`}
            >
              {message.text}
            </span>
          )}
        </div>
      </div>
    </div>
  )
}
