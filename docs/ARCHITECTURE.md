# domdrift Architecture Specification

## 1. Design Principles

`domdrift` is designed for high-throughput, low-latency, resilient web extraction with autonomous recovery from front-end changes.

Key design constraints:
- Minimal footprint: The core Go process maintains an idle memory footprint below 20MB.
- Autonomous recovery: Failures in fast selectors escalate through accessibility and visual tiers automatically.
- Cache persistence: Discovered and synthesized selectors are validated on peer elements before persistence.
- Zero runtime overhead for the happy path: When cached selectors match, extraction completes in single-digit milliseconds.

## 2. Three-Tier Extraction Protocol

### Tier 1: Fast Cached Selector Execution
- Evaluates cached CSS and XPath expressions directly via Chrome DevTools Protocol (`Runtime.evaluate` or `DOM.querySelectorAll`).
- Matches extracted values against schema constraints (non-empty, type check, optional regex).
- Execution time: < 5ms per target field.

### Tier 2: Accessibility Tree (A11y AXTree) Semantic Crawler
- Triggered when Tier 1 returns empty or invalid values.
- Retrieves the accessibility node hierarchy via CDP `Accessibility.getFullAXTree`.
- Traverses nodes looking for accessible roles (`heading`, `button`, `cell`, `staticText`) and matching names or labels.
- Handles relative semantic navigation: locates label nodes (such as "Price" or "SKU") and retrieves proximate sibling or child value nodes.

### Tier 3: Multimodal Visual Grounding and Auto-Patching
- Triggered when both Tier 1 and Tier 2 fail due to major layout or markup redesign.
- Captures a viewport screenshot using CDP `Page.captureScreenshot` (WebP or PNG format).
- Evaluates target element coordinates using vision grounding:
  - Vision Language Model (VLM): Prompts multimodal vision models with screenshot and target descriptions to obtain bounding box or center pixel coordinates (x, y).
  - Deterministic Visual Layout Heuristic: When offline or unconfigured, uses CDP bounding box quads and text layout analysis to identify element positions.
- Resolves coordinates to DOM node ID using CDP `DOM.getNodeForLocation`.
- Invokes the selector synthesizer to generate a stable selector path.

## 3. Minimal Stable Selector Synthesis

Given a resolved DOM node, the synthesizer walks up the DOM hierarchy and scores candidate selectors using an entropy-weighted stability metric:

$$S_{\text{stable}}(\pi) = w_{\text{attr}} \cdot \mathbb{I}(\text{data-testid} \in \pi) + w_{\text{role}} \cdot \mathbb{I}(\text{aria-role} \in \pi) - w_{\text{hash}} \cdot H(\text{classes}) - w_{\text{depth}} \cdot k$$

Where:
- $H(\text{classes})$ calculates the Shannon entropy across class tokens, penalizing machine-generated class hashes (such as `css-175oi2r` or `_x7K9`).
- $w_{\text{attr}}$ rewards stable data attributes like `data-testid`, `data-component`, `data-id`.
- $w_{\text{role}}$ rewards semantic ARIA roles.
- $w_{\text{depth}} \cdot k$ penalizes deep brittle selector chains.

The synthesized selector is checked against sibling cards in the document to ensure correct cardinality, then committed to `selectors.cache.json`.

## 4. Export Pipelines & Telemetry

- Streaming JSONL: Appends structured records with metadata (timestamp, source URL, tier used, confidence score).
- Parquet: Batches records into columnar format for analytical databases.
- Webhooks: Dispatches extraction payloads with HMAC-SHA256 signatures and exponential backoff retry.
- Prometheus Metrics (`/metrics`): Exposes standard OpenMetrics gauges and counters for `domdrift_selectors_total` partitioned by tier (Tier 1/2/3), `domdrift_drift_events_total`, and process resource telemetry.

## 5. Anti-Detect Evasion Suite (Stealth v2)

To withstand enterprise bot mitigation mechanisms (Cloudflare, DataDome), `domdrift` executes `Page.addScriptToEvaluateOnNewDocument` upon tab instantiation:
- Erases `navigator.webdriver` property.
- Mocks standard `window.chrome` and `navigator.plugins`.
- Spoofs WebGL vendor and renderer strings (`Intel(R) Iris(TM) Plus Graphics 640`).
- Overrides notification permissions and language arrays to match organic browser sessions.

## 6. Multi-Card Catalog Extraction

For catalog and tabular pages containing repeated item clusters, schemas with `type: "array"` define an `itemSelector` alongside property field rules. The engine counts matching card containers, scopes DOM queries per card index, and applies self-healing routines across item boundaries.
