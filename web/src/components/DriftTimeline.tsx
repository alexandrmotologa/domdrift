import React from 'react'
import { Activity, ShieldCheck, Sparkles, AlertCircle, ArrowRight } from 'lucide-react'

export interface DriftEvent {
  timestamp: string
  url: string
  field: string
  oldSelector: string
  newSelector: string
  recoveryTier: number
  confidence: number
  durationMs: number
  reason: string
}

interface DriftTimelineProps {
  events: DriftEvent[]
}

export const DriftTimeline: React.FC<DriftTimelineProps> = ({ events }) => {
  return (
    <div className="bg-slate-900/70 border border-slate-800 rounded-xl p-5 shadow-lg backdrop-blur">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <Activity className="w-4 h-4 text-emerald-400" />
          <h3 className="font-semibold text-slate-100 text-sm">Self-Healing Drift Timeline</h3>
        </div>
        <span className="text-xs font-mono px-2 py-0.5 rounded bg-slate-800 text-slate-400">
          {events.length} auto-repairs recorded
        </span>
      </div>

      {events.length === 0 ? (
        <div className="text-center py-8 text-slate-500 text-xs">
          <ShieldCheck className="w-8 h-8 text-slate-600 mx-auto mb-2" />
          No selector drift detected yet. Fast Tier 1 cached selectors operating stably.
        </div>
      ) : (
        <div className="space-y-3 max-h-[340px] overflow-y-auto pr-1">
          {events.map((evt, idx) => (
            <div
              key={idx}
              className="p-3 rounded-lg bg-slate-950/60 border border-slate-800/80 hover:border-slate-700 transition"
            >
              <div className="flex items-center justify-between text-xs mb-1.5">
                <span className="font-semibold text-emerald-400 flex items-center gap-1.5">
                  <Sparkles className="w-3.5 h-3.5 text-purple-400" />
                  {evt.field}
                </span>
                <div className="flex items-center gap-2">
                  <span
                    className={`font-mono text-[10px] px-2 py-0.5 rounded-full ${
                      evt.recoveryTier === 2
                        ? 'bg-amber-500/10 text-amber-300 border border-amber-500/20'
                        : 'bg-purple-500/10 text-purple-300 border border-purple-500/20'
                    }`}
                  >
                    Tier {evt.recoveryTier} {evt.recoveryTier === 2 ? 'A11y' : 'VLM Grounding'}
                  </span>
                  <span className="text-[11px] font-mono text-slate-400">{evt.durationMs}ms</span>
                </div>
              </div>

              {/* Selector Transition Diff */}
              <div className="flex items-center gap-2 font-mono text-[11px] my-2 p-2 rounded bg-slate-900 border border-slate-800 overflow-x-auto">
                <span className="text-rose-400/90 line-through truncate max-w-[45%]">
                  {evt.oldSelector || '(initial)'}
                </span>
                <ArrowRight className="w-3 h-3 text-slate-500 shrink-0" />
                <span className="text-emerald-300 font-semibold truncate max-w-[50%]">
                  {evt.newSelector}
                </span>
              </div>

              <div className="flex items-center justify-between text-[11px] text-slate-400">
                <span className="truncate max-w-[80%]">{evt.reason}</span>
                <span className="font-mono text-emerald-400/90">
                  {(evt.confidence * 100).toFixed(0)}% conf
                </span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
