import React, { useState, useEffect } from 'react'
import { Sparkles, Activity, Shield, Terminal, Zap, Radio } from 'lucide-react'
import { LiveBrowserView } from './components/LiveBrowserView.tsx'
import { DriftTimeline, DriftEvent } from './components/DriftTimeline.tsx'
import { SchemaEditor } from './components/SchemaEditor.tsx'
import { SelectorHealth, CachedSelectorEntry } from './components/SelectorHealth.tsx'
import { ExtractionData } from './components/ExtractionData.tsx'

export default function App() {
  const [currentUrl, setCurrentUrl] = useState('')
  const [loading, setLoading] = useState(false)
  const [lastRecord, setLastRecord] = useState<any>(null)
  const [driftEvents, setDriftEvents] = useState<DriftEvent[]>([])
  const [cacheData, setCacheData] = useState<Record<string, Record<string, CachedSelectorEntry>>>({})
  const [sseConnected, setSseConnected] = useState(false)

  // Fetch initial cache data and drift history
  const fetchStatus = async () => {
    try {
      const [cacheRes, driftRes] = await Promise.all([
        fetch('/api/cache'),
        fetch('/api/drift-history')
      ])
      if (cacheRes.ok) {
        const c = await cacheRes.json()
        setCacheData(c || {})
      }
      if (driftRes.ok) {
        const d = await driftRes.json()
        setDriftEvents(d || [])
      }
    } catch (e) {
      console.error('Failed fetching status:', e)
    }
  }

  useEffect(() => {
    fetchStatus()

    // Setup Server-Sent Events stream
    const events = new EventSource('/api/events')

    events.onopen = () => {
      setSseConnected(true)
    }

    events.onerror = () => {
      setSseConnected(false)
    }

    events.addEventListener('drift_event', (e: MessageEvent) => {
      try {
        const eventData = JSON.parse(e.data)
        setDriftEvents((prev) => [eventData, ...prev])
        fetchStatus()
      } catch (err) {
        console.error(err)
      }
    })

    events.addEventListener('extraction_record', (e: MessageEvent) => {
      try {
        const record = JSON.parse(e.data)
        setLastRecord(record)
      } catch (err) {
        console.error(err)
      }
    })

    events.addEventListener('cache_reset', () => {
      setCacheData({})
      setDriftEvents([])
    })

    return () => {
      events.close()
    }
  }, [])

  const handleRunExtraction = async (url: string, schema: any) => {
    setLoading(true)
    setCurrentUrl(url)
    try {
      const res = await fetch('/api/extract', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url, schema })
      })

      if (res.ok) {
        const record = await res.json()
        setLastRecord(record)
        fetchStatus()
      } else {
        const errText = await res.text()
        alert(`Extraction failed: ${errText}`)
      }
    } catch (e: any) {
      alert(`Network error: ${e.message}`)
    } finally {
      setLoading(false)
    }
  }

  const handleResetCache = async () => {
    try {
      await fetch('/api/cache/reset', { method: 'POST' })
      setCacheData({})
      setDriftEvents([])
    } catch (e) {
      console.error(e)
    }
  }

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100 flex flex-col">
      {/* Top Header Bar */}
      <header className="border-b border-slate-800 bg-slate-900/60 backdrop-blur sticky top-0 z-50 px-6 py-3.5 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 rounded-lg bg-emerald-500/10 border border-emerald-500/30 flex items-center justify-center">
            <Zap className="w-4 h-4 text-emerald-400" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h1 className="font-bold text-base tracking-tight text-slate-100">domdrift</h1>
              <span className="font-mono text-[10px] px-1.5 py-0.5 rounded bg-slate-800 text-slate-400 border border-slate-700">
                v1.0.0
              </span>
            </div>
            <p className="text-[11px] text-slate-400 font-medium">
              Self-Healing Web Extraction Engine & Multimodal Visual Grounding
            </p>
          </div>
        </div>

        <div className="flex items-center gap-4 text-xs font-mono">
          <div className="flex items-center gap-1.5 px-2.5 py-1 rounded bg-slate-900 border border-slate-800 text-slate-300">
            <Radio className={`w-3 h-3 ${sseConnected ? 'text-emerald-400 animate-pulse' : 'text-slate-500'}`} />
            <span>SSE {sseConnected ? 'Streaming' : 'Connecting'}</span>
          </div>
          <div className="px-2.5 py-1 rounded bg-slate-900 border border-slate-800 text-slate-400">
            Memory Target: <span className="text-emerald-400">&lt; 20MB RAM</span>
          </div>
        </div>
      </header>

      {/* Main Studio Workbench Grid */}
      <main className="flex-1 p-6 grid grid-cols-1 lg:grid-cols-12 gap-6 max-w-[1680px] w-full mx-auto">
        {/* Left Column: Schema Designer & Cache Health (5 cols) */}
        <div className="lg:col-span-5 flex flex-col gap-6">
          <SchemaEditor onRun={handleRunExtraction} loading={loading} />
          <SelectorHealth cacheData={cacheData} onResetCache={handleResetCache} />
        </div>

        {/* Right Column: Live Browser View, Data Table, Drift Timeline (7 cols) */}
        <div className="lg:col-span-7 flex flex-col gap-6">
          <div className="h-[460px]">
            <LiveBrowserView
              url={currentUrl || lastRecord?.url || ''}
              lastUpdated={lastRecord?.timestamp || null}
              tiersUsed={lastRecord?.tiersUsed || {}}
            />
          </div>
          <ExtractionData record={lastRecord} />
          <DriftTimeline events={driftEvents} />
        </div>
      </main>
    </div>
  )
}
