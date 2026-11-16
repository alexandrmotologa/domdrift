import React from 'react'
import { Shield, RotateCcw, Cpu, Check, AlertCircle } from 'lucide-react'

export interface CachedSelectorEntry {
  selector: string
  tier: number
  confidence: number
  stability: number
  lastSuccess: string
  healCount: number
}

interface SelectorHealthProps {
  cacheData: Record<string, Record<string, CachedSelectorEntry>>
  onResetCache: () => void
}

export const SelectorHealth: React.FC<SelectorHealthProps> = ({ cacheData, onResetCache }) => {
  const allEntries: { domain: string; field: string; data: CachedSelectorEntry }[] = []

  Object.entries(cacheData || {}).forEach(([domain, fields]) => {
    Object.entries(fields || {}).forEach(([field, data]) => {
      allEntries.push({ domain, field, data })
    })
  })

  const tier1Count = allEntries.filter((e) => e.data.tier === 1).length
  const tier2Count = allEntries.filter((e) => e.data.tier === 2).length
  const tier3Count = allEntries.filter((e) => e.data.tier === 3).length
  const total = allEntries.length || 1

  return (
    <div className="bg-slate-900/70 border border-slate-800 rounded-xl p-5 shadow-lg backdrop-blur">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <Shield className="w-4 h-4 text-emerald-400" />
          <h3 className="font-semibold text-slate-100 text-sm">Learned Selector Health & Cache</h3>
        </div>
        <button
          onClick={onResetCache}
          className="text-xs text-slate-400 hover:text-rose-400 flex items-center gap-1 transition"
          title="Reset Selector Cache"
        >
          <RotateCcw className="w-3 h-3" />
          Reset Cache
        </button>
      </div>

      {/* Health Distribution Bar */}
      <div className="mb-4">
        <div className="flex items-center justify-between text-xs text-slate-400 mb-1.5 font-mono">
          <span>Tier Distribution</span>
          <span>{allEntries.length} Learned Selectors</span>
        </div>
        <div className="h-2 w-full bg-slate-800 rounded-full overflow-hidden flex">
          <div
            style={{ width: `${(tier1Count / total) * 100}%` }}
            className="bg-emerald-500 h-full transition-all duration-500"
            title={`Tier 1 (Cached CSS): ${tier1Count}`}
          />
          <div
            style={{ width: `${(tier2Count / total) * 100}%` }}
            className="bg-amber-500 h-full transition-all duration-500"
            title={`Tier 2 (A11y Tree): ${tier2Count}`}
          />
          <div
            style={{ width: `${(tier3Count / total) * 100}%` }}
            className="bg-purple-500 h-full transition-all duration-500"
            title={`Tier 3 (VLM Vision): ${tier3Count}`}
          />
        </div>
        <div className="flex items-center justify-between text-[11px] text-slate-400 mt-2 font-mono">
          <span className="flex items-center gap-1">
            <span className="w-2 h-2 rounded-full bg-emerald-500" /> Tier 1: {tier1Count}
          </span>
          <span className="flex items-center gap-1">
            <span className="w-2 h-2 rounded-full bg-amber-500" /> Tier 2: {tier2Count}
          </span>
          <span className="flex items-center gap-1">
            <span className="w-2 h-2 rounded-full bg-purple-500" /> Tier 3: {tier3Count}
          </span>
        </div>
      </div>

      {/* Cached Selectors Table */}
      {allEntries.length === 0 ? (
        <div className="text-center py-6 text-slate-500 text-xs">
          Selector cache is empty. Run an extraction to learn stable DOM paths.
        </div>
      ) : (
        <div className="space-y-2 max-h-[220px] overflow-y-auto pr-1">
          {allEntries.map((item, idx) => (
            <div
              key={idx}
              className="p-2.5 rounded bg-slate-950/60 border border-slate-800/80 flex items-center justify-between text-xs"
            >
              <div className="min-w-0 flex-1 mr-3">
                <div className="flex items-center gap-2">
                  <span className="font-semibold text-slate-200">{item.field}</span>
                  <span className="text-[10px] text-slate-500 font-mono">({item.domain})</span>
                </div>
                <div className="font-mono text-[11px] text-slate-400 truncate mt-0.5">
                  {item.data.selector}
                </div>
              </div>
              <div className="flex items-center gap-2 shrink-0">
                <span
                  className={`font-mono text-[10px] px-1.5 py-0.5 rounded ${
                    item.data.tier === 1
                      ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                      : item.data.tier === 2
                      ? 'bg-amber-500/10 text-amber-300 border border-amber-500/20'
                      : 'bg-purple-500/10 text-purple-300 border border-purple-500/20'
                  }`}
                >
                  T{item.data.tier}
                </span>
                <span className="text-[10px] font-mono text-slate-400">
                  {item.data.healCount > 0 ? `${item.data.healCount} heals` : 'stable'}
                </span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
