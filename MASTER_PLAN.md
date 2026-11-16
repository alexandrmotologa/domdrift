# domdrift: Architecture and Engineering Master Plan
> Self-Healing Schema-Driven Web Extraction Engine & Anti-Drift Scraper with Multimodal Visual Grounding
> Target: High-Performance Single-Binary Go Engine (< 20MB RAM) with Embedded React 19 Extraction Studio & Chrome DevTools Protocol Bridge

---

## 1. Problem Statement and Motivation

### 1.1 The Fragility and High Maintenance Cost of Traditional Web Scraping
Web data extraction is foundational for competitive market intelligence, AI training corpora, e-commerce price monitoring, and financial research. However, traditional scraping infrastructure suffers from continuous, expensive operational failures:
1. **CSS/XPath Selector Brittleness**: Web scrapers rely on hardcoded DOM selectors (e.g., `div.product-card > span.price-tag`). Modern frontend frameworks (React, Next.js, Vue) frequently employ dynamic class hashing (`style_price__x7K9`), obfuscated Tailwind utility classes, or periodic UI redesigns that instantly break selectors.
2. **High Human Maintenance Overhead**: In large-scale scraping operations, data engineering teams spend 40% to 60% of their sprints identifying broken scrapers, re-inspecting pages via browser DevTools, and updating selector scripts.
3. **Silent Data Ingestion Failures**: When a selector breaks, scrapers often emit `null` or empty strings rather than raising hard errors, silently corrupting downstream data pipelines and machine learning feature stores for days or weeks before detection.
4. **Complex Client-Side SPAs and Infinite Scrolling**: Modern single-page applications defer rendering to JavaScript execution, dynamic hydration, shadow DOM roots, and virtualized lists, defeating lightweight HTTP scraping scripts.

### 1.2 The domdrift Solution
`domdrift` is an autonomous, self-healing web extraction engine written in Go. Rather than coupling extraction to rigid DOM paths, `domdrift` executes a three-tier hierarchical extraction and recovery protocol:
- **Schema-Driven Extraction**: Users define target data entities using standard Zod or JSON Schema definitions (e.g., product titles, current prices, stock status, technical specifications).
- **Tier 1 (Fast Cached Selectors)**: Executes ultra-fast native CDP queries using cached selector paths with sub-5ms extraction times.
- **Tier 2 (Accessibility & Semantic Proximity)**: If Tier 1 fails, inspects the Chrome Accessibility Tree (A11y AXTree), matching elements by accessible role, label, and parent-child structural proximity.
- **Tier 3 (Multimodal Visual Grounding & Auto-Patching)**: When major redesigns obliterate DOM structure, captures a high-resolution viewport screenshot and prompts a Vision-Language Model (VLM) with visual coordinates. Upon localizing the target element, `domdrift` automatically synthesizes the new minimal stable selector, validates it against peer elements, updates the local selector cache, and resumes streaming extraction with zero downtime.
- **Embedded React 19 Extraction Studio**: Provides a live visual workbench to define schemas, preview extractions, and inspect selector drift histories.

---

## 2. High-Level System Architecture

```
       +-------------------------------------------------------------+
       |               Target Web Page / Single Page App             |
       |           (E-Commerce, Job Portal, Regulatory Site)         |
       +-------------------------------------------------------------+
                                      |
                     Headless Chrome DevTools Protocol (CDP)
                                      v
       +-------------------------------------------------------------+
       |                      domdrift Core Engine                   |
       |                    (High-Performance Go CLI)                |
       |                                                             |
       |  +---------------------+        +------------------------+  |
       |  | CDP Connection &    |------->| Schema Extraction      |  |
       |  | Navigation Manager  |        | Orchestrator           |  |
       |  +---------------------+        +------------------------+  |
       |                                              |              |
       |                                              v              |
       |  +-------------------------------------------------------+  |
       |  |               Three-Tier Extraction Pipeline          |  |
       |  |                                                       |  |
       |  |  Tier 1: Fast Cached Selector (CSS / XPath / Attr)    |  |
       |  |          | (On Drift / Missing Data)                  |  |
       |  |          v                                            |  |
       |  |  Tier 2: Accessibility Tree (A11y AXTree Semantics)   |  |
       |  |          | (On Structural Overhaul)                   |  |
       |  |          v                                            |  |
       |  |  Tier 3: Multimodal Visual Grounding (VLM)            |  |
       |  |          - Coordinate Extraction [x, y]               |  |
       |  |          - DOM Node Resolution at Point               |  |
       |  |          - Minimal Stable Selector Synthesizer        |  |
       |  +-------------------------------------------------------+  |
       |                               |                             |
       |                               v                             |
       |  +-------------------------------------------------------+  |
       |  |             Cache Auto-Updater & Validator            |  |
       |  |  - Validates Synthesized Selector on Peer Elements    |  |
       |  |  - Commits Self-Healed Path to `selectors.cache`      |  |
       |  +-------------------------------------------------------+  |
       |                               |                             |
       +-------------------------------|-----------------------------+
                                       |
                                       v
       +-------------------------------------------------------------+
       |                  Validated Structured Output                |
       |  1. Streaming Output (JSONL, Parquet, Webhook, SQLite)      |
       |  2. Async Drift Alert Dispatched (Slack / PagerDuty)        |
       |  3. Embedded React 19 Studio (Local HTTP :8282)             |
       +-------------------------------------------------------------+
```

---

## 3. Hierarchical Extraction and Self-Healing Protocol

```
+-----------------------------------+-----------------------------------------------------------------+
| Recovery Tier                     | Operational Strategy                                            |
+-----------------------------------+-----------------------------------------------------------------+
| Tier 1: Cached Fast Path          | Evaluates cached CSS/XPath query via CDP Runtime.evaluate in    |
|                                   | < 5ms. If non-empty and schema valid, returns immediately.      |
| Tier 2: A11y Semantic Proximity   | Scans Accessibility tree for roles (heading, button, price).   |
|                                   | Traverses relative semantic siblings (e.g. text next to "$").   |
| Tier 3: Visual Grounding (VLM)    | Captures viewport WebP; VLM identifies pixel coordinates        |
|                                   | $(x, y)$ of target data. CDP resolves `DOM.getNodeForLocation`. |
| Auto-Patching & Selector Synthesis| Traverses upwards from resolved node; computes shortest unique  |
|                                   | selector invariant to dynamic hashes; updates local cache.      |
+-----------------------------------+-----------------------------------------------------------------+
```

### 3.1 Selector Stability Formulation ($S_{\text{stable}}$)
Let $\pi = (n_1, n_2, \dots, n_k)$ be a candidate DOM traversal path. The stability score $S_{\text{stable}}$ balances uniqueness against vulnerability to framework rebuilds:

$$S_{\text{stable}}(\pi) = w_{\text{attr}} \cdot \mathbb{I}(\text{data-testid} \in \pi) + w_{\text{role}} \cdot \mathbb{I}(\text{aria-role} \in \pi) - w_{\text{hash}} \cdot H(\text{classes}) - w_{\text{depth}} \cdot k$$

Where $H(\text{classes})$ measures entropy/randomness of CSS class tokens (detecting hashes like `_x7K9`), favoring semantic attributes (`data-*`, `aria-*`, semantic tags) over brittle deep cascades.

---

## 4. Repository Structure

```
domdrift/
├── cmd/
│   └── domdrift/
│       └── main.go                 # CLI entry point, subcommands: extract, crawl, heal, studio
├── internal/
│   ├── config/
│   │   ├── config.go               # YAML configuration loader for targets, schedules, thresholds
│   │   └── schema.go               # Target Zod/JSON schema definitions and extraction rules
│   ├── browser/
│   │   ├── cdp.go                  # Chrome DevTools Protocol WebSocket client
│   │   ├── launcher.go             # Headless Chrome process manager with stealth flags
│   │   └── session.go              # Tab lifecycle, network interception, cookie jar
│   ├── extractor/
│   │   ├── tier1_fast.go           # Fast cached CSS/XPath selector executor
│   │   ├── tier2_a11y.go           # Accessibility tree (AXTree) crawler and role matcher
│   │   └── orchestrator.go         # Extraction fallback state machine
│   ├── healing/
│   │   ├── vision.go               # VLM client for visual coordinate grounding
│   │   ├── resolver.go             # CDP coordinate to DOM node mapper (getNodeForLocation)
│   │   ├── synthesizer.go          # Minimal stable selector generator and entropy scorer
│   │   └── cache.go                # Thread-safe persistent selector cache (JSON / SQLite)
│   ├── export/
│   │   ├── jsonl.go                # High-throughput streaming JSONL writer
│   │   ├── parquet.go              # Columnar Parquet writer for analytics pipelines
│   │   └── webhook.go              # Push delivery client with HMAC signatures
│   └── server/
│       ├── api.go                  # REST API for schema mapping and live extractions
│       ├── embed.go                # Embedded React 19 studio assets
│       └── sse.go                  # Real-time event stream for extraction monitoring
├── web/
│   ├── src/
│   │   ├── App.tsx                 # React 19 root application
│   │   ├── components/
│   │   │   ├── LiveBrowserView.tsx # Interactive iframe/canvas with highlighted extracted nodes
│   │   │   ├── DriftTimeline.tsx   # Visual history of selector breaks and auto-repairs
│   │   │   ├── SchemaEditor.tsx    # In-browser schema definition and field mapper
│   │   │   ├── SelectorHealth.tsx  # Confidence gauge and stability score breakdown
│   │   │   └── ExtractionData.tsx  # Real-time extracted record table
│   │   └── pages/
│   │       └── Studio.tsx          # Main visual extraction workbench
│   ├── package.json
│   ├── tsconfig.json
│   └── vite.config.ts
├── examples/
│   ├── domdrift.yaml               # Standard extraction configuration
│   ├── schemas/
│   │   ├── ecommerce_product.json  # Sample product pricing & availability schema
│   │   └── job_posting.json        # Sample recruitment posting schema
│   └── sample_page.html            # Test HTML fixture simulating CSS class redesign
├── Makefile
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

---

## 5. Implementation Roadmap

### Phase 1: Go Engine Foundation and CDP Headless Launcher
- Initialize Go module with minimal dependencies.
- Build native CDP WebSocket client in `internal/browser/cdp.go` avoiding heavy Node.js runtimes.
- Support headless Chrome lifecycle management with stealth flags (disabling automation flags, configuring viewport).

### Phase 2: Schema Parser and Tier 1 Cached Selector Pipeline
- Implement target schema loader in `internal/config/schema.go` supporting typed primitives, arrays, and nested objects.
- Build `tier1_fast.go` querying the DOM via `Runtime.evaluate` and `DOM.querySelector` in $< 5\text{ms}$.

### Phase 3: Tier 2 Accessibility Tree Semantic Crawler
- Implement `tier2_a11y.go` requesting `Accessibility.getFullAXTree` via CDP.
- Match elements by role, accessible name, and nearby structural label tokens (e.g. identifying currency symbols preceding numbers).

### Phase 4: Tier 3 Multimodal Visual Grounding and Auto-Patching
- Implement screenshot capture in `internal/healing/vision.go` encoding viewport to compressed WebP.
- Integrate VLM client passing target field descriptions and obtaining relative pixel coordinates.
- Call `DOM.getNodeForLocation` to resolve DOM node and synthesize the minimal stable selector in `synthesizer.go`.
- Validate synthesized selector against peer cards and commit to `selectors.cache.json`.

### Phase 5: Streaming Exporters and Data Delivery
- Implement high-speed streaming JSONL and Parquet output.
- Build webhook delivery pipeline with exponential retries and HMAC authentication.

### Phase 6: Embedded React 19 Extraction Studio
- Develop visual workbench in `web/` featuring live browser preview, interactive element selector, drift history timeline, and schema editor.
- Embed frontend build artifacts into the Go binary via `//go:embed` and serve locally on `:8282`.

### Phase 7: Verification, Test Suite, and Documentation
- Write end-to-end integration tests using `sample_page.html` with intentionally mutated CSS class names to verify automatic Tier 3 recovery.
- Benchmark memory footprint verifying idle usage remains strictly under 20MB RAM.
- Author humanized README with setup guides, Docker container definitions, and schema configuration examples.
