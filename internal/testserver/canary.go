package testserver

import (
	"fmt"
	"net/http"
)

// StartCanaryServer launches an HTTP test server serving mutated DOM fixtures.
func StartCanaryServer(port int) (*http.Server, error) {
	mux := http.NewServeMux()

	mux.HandleFunc("/v1", handleV1Clean)
	mux.HandleFunc("/v2", handleV2Hashed)
	mux.HandleFunc("/v3", handleV3Redesigned)
	mux.HandleFunc("/v1/catalog", handleV1Catalog)
	mux.HandleFunc("/v2/catalog", handleV2Catalog)
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","server":"domdrift-canary"}`))
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		_ = srv.ListenAndServe()
	}()

	return srv, nil
}

func handleV1Clean(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Canary Store - V1 Standard</title>
  <style>
    body { font-family: system-ui, sans-serif; padding: 40px; background: #f8fafc; }
    .product-card { background: #fff; padding: 24px; border-radius: 12px; max-width: 480px; box-shadow: 0 4px 6px -1px rgba(0,0,0,0.1); }
    .product-title { font-size: 24px; font-weight: 700; color: #0f172a; margin-bottom: 8px; }
    .product-price { font-size: 22px; font-weight: 600; color: #16a34a; margin-bottom: 12px; display: inline-block; }
    .stock-status { font-size: 14px; font-weight: 500; color: #0284c7; background: #e0f2fe; padding: 4px 8px; border-radius: 4px; }
    .sku-code { font-size: 12px; color: #64748b; margin-top: 12px; }
  </style>
</head>
<body>
  <div class="product-card" data-testid="product-card">
    <h1 class="product-title" data-testid="product-title">AeroPro Wireless Noise-Cancelling Headphones</h1>
    <div>
      <span class="product-price" data-testid="price">$349.99</span>
      <span class="stock-status" data-testid="stock">In Stock (14 units)</span>
    </div>
    <div class="sku-code" data-testid="sku">SKU: AERO-WH-900</div>
  </div>
</body>
</html>`))
}

func handleV2Hashed(w http.ResponseWriter, r *http.Request) {
	// Simulates modern SPA framework rebuild with randomized class hashes
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Canary Store - V2 Obfuscated Dynamic Classes</title>
  <style>
    body { font-family: system-ui, sans-serif; padding: 40px; background: #f8fafc; }
    ._card_x89a { background: #fff; padding: 24px; border-radius: 12px; max-width: 480px; box-shadow: 0 4px 6px -1px rgba(0,0,0,0.1); }
    ._h1_9xK2 { font-size: 24px; font-weight: 700; color: #0f172a; margin-bottom: 8px; }
    ._pr_4mNp { font-size: 22px; font-weight: 600; color: #16a34a; margin-bottom: 12px; display: inline-block; }
    ._st_7qRt { font-size: 14px; font-weight: 500; color: #0284c7; background: #e0f2fe; padding: 4px 8px; border-radius: 4px; }
    ._sku_1aBc { font-size: 12px; color: #64748b; margin-top: 12px; }
  </style>
</head>
<body>
  <main class="_card_x89a">
    <h1 class="_h1_9xK2" role="heading">AeroPro Wireless Noise-Cancelling Headphones</h1>
    <div>
      <span class="_pr_4mNp" role="text">$349.99</span>
      <span class="_st_7qRt">In Stock (14 units)</span>
    </div>
    <div class="_sku_1aBc">SKU: AERO-WH-900</div>
  </main>
</body>
</html>`))
}

func handleV3Redesigned(w http.ResponseWriter, r *http.Request) {
	// Simulates complete layout overhaul: flex to grid, semantic tags mutated
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Canary Store - V3 Redesigned Layout</title>
  <style>
    body { font-family: system-ui, sans-serif; padding: 40px; background: #0f172a; color: #f8fafc; }
    .viewport-grid { display: grid; grid-template-columns: 1fr; max-width: 500px; gap: 16px; background: #1e293b; padding: 32px; border-radius: 16px; }
    .item-banner { font-size: 26px; font-weight: 800; color: #38bdf8; }
    .pricing-cluster { display: flex; align-items: baseline; gap: 12px; }
    .currency-val { font-size: 28px; font-weight: 700; color: #4ade80; }
    .badge-stock { background: #0369a1; color: #e0f2fe; padding: 6px 12px; border-radius: 9999px; font-size: 13px; }
  </style>
</head>
<body>
  <div class="viewport-grid">
    <div class="item-banner">AeroPro Wireless Noise-Cancelling Headphones</div>
    <div class="pricing-cluster">
      <div class="currency-val">$349.99</div>
      <div class="badge-stock">In Stock (14 units)</div>
    </div>
    <div style="font-size: 12px; color: #94a3b8;">SKU: AERO-WH-900</div>
  </div>
</body>
</html>`))
}

func handleV1Catalog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Canary Catalog - V1 Standard</title>
  <style>
    body { font-family: system-ui, sans-serif; padding: 30px; background: #f8fafc; }
    .catalog-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 20px; max-width: 1000px; }
    .product-card { background: #fff; padding: 20px; border-radius: 8px; border: 1px solid #e2e8f0; }
    .title { font-size: 18px; font-weight: bold; margin-bottom: 8px; }
    .price { font-size: 16px; color: #16a34a; font-weight: 600; }
  </style>
</head>
<body>
  <h1>Electronics Catalog</h1>
  <div class="catalog-grid">
    <div class="product-card">
      <div class="title">Quantum Pro ANC Headphones</div>
      <div class="price">$299.99</div>
    </div>
    <div class="product-card">
      <div class="title">UltraWide 4K Gaming Monitor 34"</div>
      <div class="price">$749.50</div>
    </div>
    <div class="product-card">
      <div class="title">Mechanical Wireless Keyboard RGB</div>
      <div class="price">$129.00</div>
    </div>
  </div>
</body>
</html>`))
}

func handleV2Catalog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Canary Catalog - V2 Hashed</title>
  <style>
    body { font-family: system-ui, sans-serif; padding: 30px; background: #f8fafc; }
    ._grid_98a { display: grid; grid-template-columns: repeat(3, 1fr); gap: 20px; max-width: 1000px; }
    ._card_4b2 { background: #fff; padding: 20px; border-radius: 8px; border: 1px solid #e2e8f0; }
    ._tit_87c { font-size: 18px; font-weight: bold; margin-bottom: 8px; }
    ._pr_11d { font-size: 16px; color: #16a34a; font-weight: 600; }
  </style>
</head>
<body>
  <h1>Electronics Catalog</h1>
  <div class="_grid_98a">
    <div class="_card_4b2">
      <div class="_tit_87c" role="heading">Quantum Pro ANC Headphones</div>
      <div class="_pr_11d">$299.99</div>
    </div>
    <div class="_card_4b2">
      <div class="_tit_87c" role="heading">UltraWide 4K Gaming Monitor 34"</div>
      <div class="_pr_11d">$749.50</div>
    </div>
    <div class="_card_4b2">
      <div class="_tit_87c" role="heading">Mechanical Wireless Keyboard RGB</div>
      <div class="_pr_11d">$129.00</div>
    </div>
  </div>
</body>
</html>`))
}
