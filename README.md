# domdrift

Self-healing schema-driven web extraction engine in Go with Chrome DevTools Protocol, three-tier visual grounding, and embedded React 19 extraction studio.

## Overview

Web scrapers typically fail when target applications deploy dynamic class hashes, modified component hierarchies, or redesigned layouts. `domdrift` addresses selector drift through an autonomous three-tier recovery protocol:

- Tier 1 (Cached Fast Path): Queries the DOM via cached CSS or XPath selectors through Chrome DevTools Protocol (CDP) in under 5ms.
- Tier 2 (Accessibility Tree): When cached selectors fail, parses the Chrome Accessibility Tree (AXTree) to resolve targets by role, accessible name, and semantic label proximity.
- Tier 3 (Multimodal Visual Grounding): When markup undergoes major structural redesign, captures a viewport screenshot, resolves target element coordinates using vision grounding, maps pixel locations to DOM nodes via CDP, and synthesizes a minimal stable selector invariant to dynamic class hashes.

The synthesized selector is validated across sibling elements, written to a persistent cache, and used for subsequent extractions without halting execution pipelines.

## Core Features

- High performance Go core with memory footprint below 20MB RAM in idle state.
- Pure WebSocket Chrome DevTools Protocol bridge with stealth flags and no Node.js runtime dependency.
- Stealth v2 fingerprint evasion suite masking automation flags, navigator properties, and WebGL renderer.
- Three-tier extraction protocol: Cached Selectors, AXTree Semantics, and Multimodal Visual Grounding.
- Single-entity and multi-card catalog extraction with item-level self-healing.
- Entropy-based selector synthesis minimizing fragility against randomized CSS classes.
- Streaming output formats: JSONL, columnar Parquet, and authenticated HMAC-SHA256 webhooks.
- Prometheus `/metrics` endpoint exporting live telemetry for selector health, extraction duration, and drift events.
- Built-in canary mutation test server for validating self-healing routines across DOM version mutations.
- Embedded React 19 studio interface with interactive point-and-click visual field inspection, schema designer, and live DOM telemetry.

## System Architecture

```
Target Web Application
         │
  Chrome DevTools Protocol (CDP WebSocket)
         ▼
domdrift Core Engine (Go)
  ├── Browser & Session Manager (Stealth Headless Chrome)
  ├── Schema Extraction Orchestrator
  │     ├── Tier 1: Cached Fast Path (CSS / XPath)
  │     ├── Tier 2: A11y Semantic Proximity (AXTree)
  │     └── Tier 3: Multimodal Visual Grounding (VLM + Local Heuristic)
  ├── Stable Selector Synthesizer (Entropy Scorer)
  ├── Persistent Selector Cache (JSON / SQLite)
  └── Exporters (JSONL, Parquet, Webhook)
         │
Embedded React 19 Studio (:8282) & SSE Event Stream
```

## Getting Started

### Prerequisites

- Go 1.22 or newer
- Google Chrome, Chromium, or Microsoft Edge installed locally
- Node.js 20+ and npm (only needed when rebuilding the web studio)

### Installation

```bash
git clone https://github.com/alexandrmotologa/domdrift.git
cd domdrift
go build -o bin/domdrift ./cmd/domdrift
```

### CLI Usage

Run extraction against a target URL using a schema:

```bash
domdrift extract --url https://example.com/products --schema examples/schemas/ecommerce_product.json --output out/products.jsonl
```

Start the interactive React 19 Extraction Studio:

```bash
domdrift studio --port 8282
```

Run the built-in drift canary mutation test server:

```bash
domdrift test-server --port 9090
```
