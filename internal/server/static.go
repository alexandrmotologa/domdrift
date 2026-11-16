package server

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var embeddedStudioFS embed.FS

// StaticFileHandler serves the embedded React 19 studio single page app.
type StaticFileHandler struct {
	fileServer http.Handler
	subFS      fs.FS
}

// NewStaticFileHandler initializes an HTTP handler serving embedded frontend assets.
func NewStaticFileHandler() *StaticFileHandler {
	sub, err := fs.Sub(embeddedStudioFS, "dist")
	if err != nil {
		panic(err)
	}

	return &StaticFileHandler{
		fileServer: http.FileServer(http.FS(sub)),
		subFS:      sub,
	}
}

func (h *StaticFileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		http.NotFound(w, r)
		return
	}

	trimmedPath := strings.TrimPrefix(r.URL.Path, "/")
	if trimmedPath == "" {
		trimmedPath = "index.html"
	}

	// Check if file exists in embedded filesystem
	if _, err := fs.Stat(h.subFS, trimmedPath); err == nil {
		h.fileServer.ServeHTTP(w, r)
		return
	}

	// SPA fallback: serve index.html for client-side routing
	indexFile, err := h.subFS.Open("index.html")
	if err == nil {
		defer indexFile.Close()
		stat, _ := indexFile.Stat()
		if seeker, ok := indexFile.(io.ReadSeeker); ok {
			http.ServeContent(w, r, "index.html", stat.ModTime(), seeker)
			return
		}
	}

	http.NotFound(w, r)
}
