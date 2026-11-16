import React, { useState } from 'react'
import { Eye, RefreshCw, Globe, CheckCircle2, AlertTriangle } from 'lucide-react'

interface LiveBrowserViewProps {
  url: string
  lastUpdated: string | null
  tiersUsed: Record<string, number>
}

export const LiveBrowserView: React.FC<LiveBrowserViewProps> = ({ url, lastUpdated, tiersUsed }) => {
  const [imgKey, setImgKey] = useState(Date.now())
  const [loading, setLoading] = useState(false)
  const [hasError, setHasError] = useState(false)

  const refreshScreenshot = () => {
    setLoading(true)
    setHasError(false)
    setImgKey(Date.now())
    setTimeout(() => setLoading(false), 600)
  }

  return (
    <div className="bg-slate-900/70 border border-slate-800 rounded-xl overflow-hidden flex flex-col h-full shadow-lg backdrop-blur">
      {/* Browser Bar */}
      <div className="bg-slate-900 border-b border-slate-800 px-4 py-2.5 flex items-center justify-between gap-3 text-xs">
        <div className="flex items-center gap-2 flex-1 min-w-0">
          <Globe className="w-3.5 h-3.5 text-emerald-400 shrink-0" />
          <span className="font-mono text-slate-300 truncate">{url || 'No active URL loaded'}</span>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 font-mono text-[11px] border border-emerald-500/20">
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
            CDP Headless (1920x1080)
          </span>
          <button
            onClick={refreshScreenshot}
            disabled={loading}
            className="p-1 hover:bg-slate-800 text-slate-400 hover:text-slate-200 rounded transition"
            title="Refresh Viewport"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
          </button>
        </div>
      </div>

      {/* Viewport Canvas */}
      <div className="relative flex-1 bg-slate-950 flex items-center justify-center min-h-[360px] overflow-auto p-4">
        {!hasError ? (
          <img
            key={imgKey}
            src={`/api/screenshot?t=${imgKey}`}
            onError={() => setHasError(true)}
            alt="Live Headless Chrome Viewport"
            className="max-w-full max-h-[500px] object-contain rounded border border-slate-800/80 shadow-2xl"
          />
        ) : (
          <div className="text-center p-8 max-w-sm">
            <Eye className="w-10 h-10 text-slate-600 mx-auto mb-3" />
            <p className="text-slate-400 text-sm font-medium">No live screenshot captured yet</p>
            <p className="text-slate-600 text-xs mt-1">Execute an extraction to stream the headless browser viewport.</p>
          </div>
        )}

        {/* Floating Extraction Overlay Badges */}
        {Object.keys(tiersUsed).length > 0 && (
          <div className="absolute top-6 right-6 flex flex-col gap-1.5 pointer-events-none">
            {Object.entries(tiersUsed).map(([field, tier]) => (
              <div
                key={field}
                className="px-2.5 py-1 rounded bg-slate-900/90 border border-slate-700/80 backdrop-blur shadow-md flex items-center gap-2 text-xs"
              >
                <span className="font-semibold text-slate-200">{field}:</span>
                <span
                  className={`font-mono text-[10px] px-1.5 py-0.2 rounded ${
                    tier === 1
                      ? 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/30'
                      : tier === 2
                      ? 'bg-amber-500/20 text-amber-300 border border-amber-500/30'
                      : 'bg-purple-500/20 text-purple-300 border border-purple-500/30'
                  }`}
                >
                  Tier {tier} {tier === 1 ? 'Cached' : tier === 2 ? 'A11y' : 'VLM Vision'}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
