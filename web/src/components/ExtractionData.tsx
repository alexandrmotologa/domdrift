import React, { useState } from 'react'
import { Database, Copy, Check, Table, FileJson } from 'lucide-react'

interface ExtractionDataProps {
  record: {
    url: string
    timestamp: string
    data: Record<string, any>
    tiersUsed: Record<string, number>
  } | null
}

export const ExtractionData: React.FC<ExtractionDataProps> = ({ record }) => {
  const [viewMode, setViewMode] = useState<'table' | 'json'>('table')
  const [copied, setCopied] = useState(false)

  const handleCopy = () => {
    if (!record) return
    navigator.clipboard.writeText(JSON.stringify(record, null, 2))
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  return (
    <div className="bg-slate-900/70 border border-slate-800 rounded-xl p-5 shadow-lg backdrop-blur">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <Database className="w-4 h-4 text-emerald-400" />
          <h3 className="font-semibold text-slate-100 text-sm">Extracted Structured Record</h3>
        </div>
        <div className="flex items-center gap-2">
          <div className="flex rounded bg-slate-800 p-0.5 text-xs">
            <button
              onClick={() => setViewMode('table')}
              className={`px-2 py-1 rounded flex items-center gap-1 transition ${
                viewMode === 'table' ? 'bg-slate-700 text-slate-100 font-medium' : 'text-slate-400 hover:text-slate-200'
              }`}
            >
              <Table className="w-3 h-3" /> Table
            </button>
            <button
              onClick={() => setViewMode('json')}
              className={`px-2 py-1 rounded flex items-center gap-1 transition ${
                viewMode === 'json' ? 'bg-slate-700 text-slate-100 font-medium' : 'text-slate-400 hover:text-slate-200'
              }`}
            >
              <FileJson className="w-3 h-3" /> JSON
            </button>
          </div>
          {record && (
            <button
              onClick={handleCopy}
              className="p-1.5 hover:bg-slate-800 text-slate-400 hover:text-slate-200 rounded transition"
              title="Copy JSON to Clipboard"
            >
              {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
            </button>
          )}
        </div>
      </div>

      {!record ? (
        <div className="text-center py-8 text-slate-500 text-xs">
          No extraction records generated yet. Launch an extraction to populate data.
        </div>
      ) : viewMode === 'table' ? (
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs border-collapse">
            <thead>
              <tr className="border-b border-slate-800 text-slate-400 font-mono">
                <th className="pb-2 font-medium">Field</th>
                <th className="pb-2 font-medium">Extracted Value</th>
                <th className="pb-2 font-medium text-right">Recovery Tier</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/60 font-mono">
              {Object.entries(record.data).map(([field, val]) => (
                <tr key={field} className="hover:bg-slate-800/20 transition">
                  <td className="py-2.5 font-semibold text-emerald-400">{field}</td>
                  <td className="py-2.5 text-slate-200">{String(val)}</td>
                  <td className="py-2.5 text-right">
                    <span
                      className={`text-[10px] px-2 py-0.5 rounded-full inline-block ${
                        record.tiersUsed[field] === 1
                          ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                          : record.tiersUsed[field] === 2
                          ? 'bg-amber-500/10 text-amber-300 border border-amber-500/20'
                          : 'bg-purple-500/10 text-purple-300 border border-purple-500/20'
                      }`}
                    >
                      Tier {record.tiersUsed[field]}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <pre className="bg-slate-950 p-3 rounded-lg border border-slate-800 text-[11px] font-mono text-emerald-300 overflow-x-auto max-h-[220px]">
          {JSON.stringify(record, null, 2)}
        </pre>
      )}
    </div>
  )
}
