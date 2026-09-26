package static

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist
var files embed.FS

// Frontend returns an http.Handler serving the built frontend assets with
// SPA fallback: any path that doesn't match a file serves index.html.
func Frontend() http.Handler {
	sub, err := fs.Sub(files, "dist")
	if err != nil {
		panic("static: embedded frontend not found — run 'cd web && npm run build'")
	}
	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path != "/" {
			f, openErr := sub.Open(path[1:]) // strip leading /
			if openErr != nil {
				// SPA fallback: serve index.html
				r.URL.Path = "/"
				fileServer.ServeHTTP(w, r)
				return
			}
			f.Close()
		}
		fileServer.ServeHTTP(w, r)
	})
}