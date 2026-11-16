import React, { useState, useRef } from 'react'
import { Eye, RefreshCw, Globe, Crosshair, Sparkles, Plus, Copy, Check, X } from 'lucide-react'

export interface InspectedElement {
  x: number
  y: number
  nodeId: number
  tagName: string
  text: string
  synthesizedSelector: string
  confidence: number
  attributes: Record<string, string>
}

interface LiveBrowserViewProps {
  url: string
  lastUpdated: string | null
  tiersUsed: Record<string, number>
  onAddInspectedField?: (fieldName: string, selector: string, semanticRole: string) => void
}

export const LiveBrowserView: React.FC<LiveBrowserViewProps> = ({
  url,
  lastUpdated,
  tiersUsed,
  onAddInspectedField,
}) => {
  const [imgKey, setImgKey] = useState(Date.now())
  const [loading, setLoading] = useState(false)
  const [hasError, setHasError] = useState(false)
  const [inspectMode, setInspectMode] = useState(false)
  const [clickPos, setClickPos] = useState<{ clientX: number; clientY: number; docX: number; docY: number } | null>(null)
  const [inspected, setInspected] = useState<InspectedElement | null>(null)
  const [inspecting, setInspecting] = useState(false)
  const [copied, setCopied] = useState(false)

  const imgRef = useRef<HTMLImageElement>(null)

  const refreshScreenshot = () => {
    setLoading(true)
    setHasError(false)
    setImgKey(Date.now())
    setTimeout(() => setLoading(false), 600)
  }

  const handleImageClick = async (e: React.MouseEvent<HTMLImageElement>) => {
    if (!inspectMode || !imgRef.current) return

    const rect = imgRef.current.getBoundingClientRect()
    const scaleX = 1920 / rect.width
    const scaleY = 1080 / rect.height

    const docX = Math.round((e.clientX - rect.left) * scaleX)
    const docY = Math.round((e.clientY - rect.top) * scaleY)

    setClickPos({
      clientX: e.clientX - rect.left,
      clientY: e.clientY - rect.top,
      docX,
      docY,
    })

    setInspecting(true)
    try {
      const res = await fetch('/api/inspect-point', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ x: docX, y: docY }),
      })

      if (res.ok) {
        const data = await res.json()
        setInspected(data)
      }
    } catch (err) {
      console.error('Point inspect failed:', err)
    } finally {
      setInspecting(false)
    }
  }

  const handleCopySelector = () => {
    if (!inspected) return
    navigator.clipboard.writeText(inspected.synthesizedSelector)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  const handleAddToSchema = () => {
    if (!inspected || !onAddInspectedField) return
    const defaultName = inspected.tagName === 'h1' || inspected.tagName === 'h2' ? 'title' :
                        inspected.text.includes('$') || inspected.text.includes('€') ? 'price' :
                        inspected.tagName.toLowerCase()
    const defaultRole = defaultName === 'price' ? 'price' : 'heading'
    onAddInspectedField(defaultName, inspected.synthesizedSelector, defaultRole)
    setInspected(null)
    setClickPos(null)
  }

  return (
    <div className="bg-slate-900/70 border border-slate-800 rounded-xl overflow-hidden flex flex-col h-full shadow-lg backdrop-blur relative">
      {/* Browser Bar */}
      <div className="bg-slate-900 border-b border-slate-800 px-4 py-2.5 flex items-center justify-between gap-3 text-xs z-10">
        <div className="flex items-center gap-2 flex-1 min-w-0">
          <Globe className="w-3.5 h-3.5 text-emerald-400 shrink-0" />
          <span className="font-mono text-slate-300 truncate">{url || 'No active URL loaded'}</span>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <button
            onClick={() => setInspectMode(!inspectMode)}
            className={`px-2.5 py-1 rounded flex items-center gap-1.5 font-mono text-[11px] transition ${
              inspectMode
                ? 'bg-emerald-500 text-slate-950 font-semibold shadow-md shadow-emerald-500/20'
                : 'bg-slate-800 text-slate-300 hover:bg-slate-700'
            }`}
          >
            <Crosshair className="w-3.5 h-3.5" />
            {inspectMode ? 'Inspect Active' : 'Point & Inspect'}
          </button>
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 font-mono text-[11px] border border-emerald-500/20">
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
            CDP Stealth v2
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
          <div className="relative inline-block">
            <img
              ref={imgRef}
              key={imgKey}
              src={`/api/screenshot?t=${imgKey}`}
              onClick={handleImageClick}
              onError={() => setHasError(true)}
              alt="Live Headless Chrome Viewport"
              className={`max-w-full max-h-[500px] object-contain rounded border border-slate-800/80 shadow-2xl transition ${
                inspectMode ? 'cursor-crosshair hover:border-emerald-500/50' : 'cursor-default'
              }`}
            />

            {/* Click Coordinate Pin Marker */}
            {clickPos && (
              <div
                style={{ left: clickPos.clientX, top: clickPos.clientY }}
                className="absolute -translate-x-1/2 -translate-y-1/2 pointer-events-none z-20 flex items-center justify-center"
              >
                <span className="w-4 h-4 rounded-full bg-emerald-400 animate-ping absolute opacity-75" />
                <span className="w-3 h-3 rounded-full bg-emerald-500 border-2 border-white shadow-lg" />
              </div>
            )}
          </div>
        ) : (
          <div className="text-center p-8 max-w-sm">
            <Eye className="w-10 h-10 text-slate-600 mx-auto mb-3" />
            <p className="text-slate-400 text-sm font-medium">No live screenshot captured yet</p>
            <p className="text-slate-600 text-xs mt-1">Execute an extraction to stream the headless browser viewport.</p>
          </div>
        )}

        {/* Floating Extraction Overlay Badges */}
        {Object.keys(tiersUsed).length > 0 && (
          <div className="absolute top-6 right-6 flex flex-col gap-1.5 pointer-events-none z-10">
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

        {/* Inspected Element Floating Inspector Modal */}
        {inspected && (
          <div className="absolute bottom-6 left-6 right-6 max-w-lg bg-slate-900/95 border border-emerald-500/40 rounded-xl p-4 shadow-2xl backdrop-blur z-30 animate-in fade-in slide-in-from-bottom-2">
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center gap-1.5 text-xs font-semibold text-emerald-400">
                <Sparkles className="w-4 h-4" />
                <span>Inspected DOM Element</span>
                <span className="font-mono text-[11px] text-slate-400 font-normal">
                  &lt;{inspected.tagName}&gt; at [{inspected.x}, {inspected.y}]
                </span>
              </div>
              <button
                onClick={() => setInspected(null)}
                className="p-1 hover:bg-slate-800 text-slate-400 hover:text-slate-200 rounded transition"
              >
                <X className="w-3.5 h-3.5" />
              </button>
            </div>

            <div className="text-xs text-slate-300 font-medium mb-2 truncate">
              "{inspected.text || '(empty text or container)'}"
            </div>

            <div className="flex items-center gap-2 p-2 rounded bg-slate-950 border border-slate-800 font-mono text-[11px] text-emerald-300 mb-3">
              <span className="truncate flex-1">{inspected.synthesizedSelector}</span>
              <button
                onClick={handleCopySelector}
                className="p-1 hover:bg-slate-800 text-slate-400 hover:text-emerald-400 rounded transition"
                title="Copy Selector"
              >
                {copied ? <Check className="w-3 h-3 text-emerald-400" /> : <Copy className="w-3 h-3" />}
              </button>
            </div>

            <div className="flex items-center justify-between pt-1 border-t border-slate-800/80">
              <span className="text-[11px] font-mono text-slate-400">
                Stability: {(inspected.confidence * 100).toFixed(0)}%
              </span>
              <button
                onClick={handleAddToSchema}
                className="px-3 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-slate-950 font-semibold text-xs flex items-center gap-1.5 shadow transition"
              >
                <Plus className="w-3.5 h-3.5" />
                Add to Schema
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
