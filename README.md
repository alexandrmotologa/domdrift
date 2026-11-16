<p align="center">
  <img src="docs/images/logo.png?raw=true" alt="domdrift Logo" width="130" style="border-radius: 24px;" />
</p>

<h1 align="center">domdrift</h1>

<p align="center">
  <strong>Self-healing schema-driven web extraction engine in Go with Chrome DevTools Protocol, three-tier visual grounding, and embedded React 19 extraction studio.</strong>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go" alt="Go Version" />
  <img src="https://img.shields.io/badge/React-19-61DAFB?style=flat&logo=react" alt="React 19" />
  <img src="https://img.shields.io/badge/CDP-Native_WebSocket-4285F4?style=flat&logo=google-chrome" alt="CDP" />
  <img src="https://img.shields.io/badge/Memory-%3C_20MB_RAM-10B981?style=flat" alt="Memory Footprint" />
  <img src="https://img.shields.io/badge/Self--Healing-3--Tier-8B5CF6?style=flat" alt="Self-Healing Tier" />
</p>

---

<p align="center">
  <img src="docs/images/domdrift_demo.gif?raw=true" alt="domdrift Autonomous 3-Tier Self-Healing Extraction Demo" width="800" style="border-radius: 8px; border: 1px solid #1e293b;" />
</p>

## Overview

Web scrapers typically fail when target applications deploy dynamic class hashes, modified component hierarchies, or redesigned layouts. `domdrift` eliminates scraper maintenance through an autonomous three-tier recovery protocol:

- **Tier 1 (Cached Fast Path)**: Evaluates cached CSS and XPath queries directly via Chrome DevTools Protocol (CDP) in under 5ms.
- **Tier 2 (Accessibility Tree)**: When cached selectors fail, parses the Chrome Accessibility Tree (AXTree) to resolve targets by role, accessible name, and semantic label proximity.
- **Tier 3 (Multimodal Visual Grounding)**: When markup undergoes major structural redesign, captures a viewport screenshot, resolves target element coordinates using vision grounding, maps pixel locations to DOM nodes via CDP, and synthesizes a minimal stable selector invariant to dynamic class hashes.

The synthesized selector is validated across sibling elements, written to a persistent cache, and used for subsequent extractions without halting execution pipelines.

---

## Visual Studio & Live Execution

### Embedded React 19 Extraction Studio

The engine embeds a complete React 19 workbench for visual schema design, live DOM inspection, and drift telemetry.

<p align="center">
  <img src="docs/images/domdrift_studio.png?raw=true" alt="domdrift Embedded React 19 Studio" width="800" style="border-radius: 8px; border: 1px solid #1e293b;" />
</p>

### CLI Autonomous Self-Healing Pipeline

Terminal execution demonstrates zero-downtime recovery when transitioning across mutated page versions:

<p align="center">
  <img src="docs/images/domdrift_terminal.png?raw=true" alt="domdrift Terminal Autonomous Extraction" width="800" style="border-radius: 8px; border: 1px solid #1e293b;" />
</p>

---

## Core Features

- High performance Go core with memory footprint strictly below 20MB RAM in idle state (measured at 12MB).
- Pure WebSocket Chrome DevTools Protocol bridge with stealth flags and zero Node.js runtime dependencies.
- Stealth v2 fingerprint evasion suite masking automation flags, navigator properties, and WebGL renderer.
- Three-tier extraction protocol: Cached Selectors, AXTree Semantics, and Multimodal Visual Grounding.
- Single-entity and multi-card catalog extraction with item-level self-healing.
- Interactive point-and-click visual field inspector inside the embedded React 19 studio.
- Entropy-based selector synthesis prioritizing stable attributes and penalizing dynamic class hashes.
- Streaming output formats: JSONL, columnar Parquet, and authenticated HMAC-SHA256 webhooks.
- Prometheus `/metrics` endpoint exporting live telemetry for selector health, extraction duration, and drift events.
- Built-in canary mutation test server for validating self-healing routines across DOM version mutations.

---

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

---

## Getting Started

### Prerequisites

- Go 1.22 or newer
- Google Chrome, Chromium, or Microsoft Edge installed locally
- Node.js 20+ and npm (only needed when authoring frontend changes in `web/`)

### Installation

```bash
git clone https://github.com/alexandrmotologa/domdrift.git
cd domdrift
go build -o bin/domdrift ./cmd/domdrift
```

### CLI Usage

Extract structured data from a single URL using a target schema:

```bash
./bin/domdrift extract --url http://127.0.0.1:9090/v1 --schema examples/schemas/ecommerce_product.json --output out/products.jsonl
```

Extract an array of products from a multi-card catalog:

```bash
./bin/domdrift extract --url http://127.0.0.1:9090/v1/catalog --schema examples/schemas/catalog_products.json --output out/catalog.jsonl
```

Launch the embedded React 19 visual extraction studio:

```bash
./bin/domdrift studio --port 8282
```

Run the built-in drift canary mutation test server:

```bash
./bin/domdrift test-server --port 9090
```

Inspect live Prometheus telemetry:

```bash
curl http://127.0.0.1:8282/metrics
```

---

## Verification & Testing

Run all unit and integration tests verifying 3-tier self-healing across DOM mutations:

```bash
go test -v ./internal/...
```
