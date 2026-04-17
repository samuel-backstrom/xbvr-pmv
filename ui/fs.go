package ui

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"path"
	"strings"
)

//go:embed dist
var Assets embed.FS

func GetFileSystem(useOS bool) http.FileSystem {
	if useOS {
		return http.Dir("ui/dist")
	}

	fs, err := fs.Sub(Assets, "dist")
	if err != nil {
		log.Panic(err)
	}
	return http.FS(fs)
}

func NewSPAHandler(useOS bool) http.Handler {
	fileSystem := GetFileSystem(useOS)
	fileServer := http.FileServer(fileSystem)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqPath := path.Clean("/" + strings.TrimPrefix(r.URL.Path, "/"))
		trimmedPath := strings.TrimPrefix(reqPath, "/")

		// Let asset requests resolve normally; fallback only for client-side routes.
		if trimmedPath == "" || trimmedPath == "." {
			reqPath = "/"
		} else if !strings.Contains(path.Base(trimmedPath), ".") {
			if _, err := fileSystem.Open(trimmedPath); err != nil {
				reqPath = "/"
			}
		}

		clone := r.Clone(r.Context())
		clone.URL.Path = reqPath
		fileServer.ServeHTTP(w, clone)
	})
}
