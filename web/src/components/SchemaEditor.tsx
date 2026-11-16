import React, { useState } from 'react'
import { Play, Code2, Sparkles, Sliders, Layers } from 'lucide-react'

interface SchemaEditorProps {
  onRun: (url: string, schema: any) => void
  loading: boolean
  schemaJson: string
  onChangeSchema: (val: string) => void
}

const DEFAULT_ECOMMERCE_SCHEMA = {
  title: "EcommerceProduct",
  type: "object",
  required: ["title", "price"],
  properties: {
    title: {
      type: "string",
      description: "Product display name",
      semanticRole: "heading",
      initialSelector: ".product-title, h1.title, [data-testid='product-title']"
    },
    price: {
      type: "string",
      description: "Current product selling price",
      semanticRole: "price",
      pattern: "[$€£¥₹]?\\s*[0-9]+([.,][0-9]{2})?",
      initialSelector: ".product-price, .price-tag, [data-testid='price']"
    },
    availability: {
      type: "string",
      description: "Stock status",
      semanticRole: "status",
      initialSelector: ".stock-status, .availability, [data-testid='stock']"
    },
    sku: {
      type: "string",
      description: "Item SKU or identifier",
      semanticRole: "identifier",
      initialSelector: ".sku-code, .product-sku, [data-testid='sku']"
    }
  }
}

const DEFAULT_CATALOG_SCHEMA = {
  title: "CatalogProducts",
  type: "array",
  itemSelector: ".product-card, ._card_4b2",
  required: ["title", "price"],
  properties: {
    title: {
      type: "string",
      description: "Product name in catalog item",
      semanticRole: "heading",
      initialSelector: ".title, h2, [role='heading']"
    },
    price: {
      type: "string",
      description: "Price in catalog item",
      semanticRole: "price",
      pattern: "[$€£¥₹]?\\s*[0-9]+([.,][0-9]{2})?",
      initialSelector: ".price, span"
    }
  }
}

export const SchemaEditor: React.FC<SchemaEditorProps> = ({
  onRun,
  loading,
  schemaJson,
  onChangeSchema,
}) => {
  const [url, setUrl] = useState('http://127.0.0.1:9090/v1')
  const [jsonError, setJsonError] = useState<string | null>(null)

  const handleJsonChange = (val: string) => {
    onChangeSchema(val)
    try {
      JSON.parse(val)
      setJsonError(null)
    } catch (e: any) {
      setJsonError(e.message)
    }
  }

  const handleExecute = () => {
    try {
      const parsed = JSON.parse(schemaJson)
      onRun(url, parsed)
    } catch (e: any) {
      setJsonError(e.message)
    }
  }

  return (
    <div className="bg-slate-900/70 border border-slate-800 rounded-xl p-5 shadow-lg backdrop-blur flex flex-col h-full">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <Code2 className="w-4 h-4 text-emerald-400" />
          <h3 className="font-semibold text-slate-100 text-sm">Extraction Schema & Target</h3>
        </div>
        <div className="flex items-center gap-1.5 text-xs text-slate-400">
          <span>Canary:</span>
          {['v1', 'v2', 'v3', 'v1/catalog', 'v2/catalog'].map((v) => (
            <button
              key={v}
              type="button"
              onClick={() => {
                setUrl(`http://127.0.0.1:9090/${v}`)
                if (v.includes('catalog')) {
                  onChangeSchema(JSON.stringify(DEFAULT_CATALOG_SCHEMA, null, 2))
                } else {
                  onChangeSchema(JSON.stringify(DEFAULT_ECOMMERCE_SCHEMA, null, 2))
                }
              }}
              className="px-1.5 py-0.5 rounded bg-slate-800 hover:bg-slate-700 text-emerald-400 font-mono text-[10px] transition"
            >
              /{v}
            </button>
          ))}
        </div>
      </div>

      {/* Target URL Input */}
      <div className="mb-4">
        <label className="block text-xs font-medium text-slate-400 mb-1.5">Target Web Page URL</label>
        <input
          type="text"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          placeholder="http://127.0.0.1:9090/v1 or https://..."
          className="w-full bg-slate-950 border border-slate-800 focus:border-emerald-500 rounded-lg px-3 py-2 text-xs font-mono text-slate-200 outline-none transition"
        />
      </div>

      {/* Schema JSON Editor */}
      <div className="flex-1 flex flex-col min-h-[220px]">
        <div className="flex items-center justify-between mb-1.5">
          <label className="text-xs font-medium text-slate-400 flex items-center gap-2">
            <span>Target Schema (Single Entity or Multi-Card Array)</span>
          </label>
          {jsonError && <span className="text-[11px] text-rose-400 font-mono">{jsonError}</span>}
        </div>
        <textarea
          value={schemaJson}
          onChange={(e) => handleJsonChange(e.target.value)}
          rows={11}
          className="w-full flex-1 bg-slate-950 border border-slate-800 focus:border-emerald-500 rounded-lg p-3 text-[11px] font-mono text-slate-300 outline-none resize-none transition"
          spellCheck={false}
        />
      </div>

      {/* Run Action Button */}
      <button
        onClick={handleExecute}
        disabled={loading || !!jsonError}
        className="mt-4 w-full bg-emerald-500 hover:bg-emerald-600 disabled:opacity-50 text-slate-950 font-semibold py-2.5 px-4 rounded-lg flex items-center justify-center gap-2 text-xs transition shadow-lg shadow-emerald-500/10 cursor-pointer disabled:cursor-not-allowed"
      >
        <Play className={`w-3.5 h-3.5 fill-current ${loading ? 'animate-pulse' : ''}`} />
        {loading ? 'Extracting via CDP & Healing...' : 'Run Extraction & Self-Heal'}
      </button>
    </div>
  )
}
