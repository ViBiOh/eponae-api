package main

import (
	"flag"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/ViBiOh/auth/pkg/auth"
	"github.com/ViBiOh/auth/pkg/ident/basic"
	identService "github.com/ViBiOh/auth/pkg/ident/service"
	"github.com/ViBiOh/eponae-api/pkg/reading"
	"github.com/ViBiOh/eponae-api/pkg/readingtag"
	"github.com/ViBiOh/eponae-api/pkg/tag"
	"github.com/ViBiOh/httputils/v3/pkg/alcotest"
	"github.com/ViBiOh/httputils/v3/pkg/cors"
	"github.com/ViBiOh/httputils/v3/pkg/crud"
	"github.com/ViBiOh/httputils/v3/pkg/db"
	"github.com/ViBiOh/httputils/v3/pkg/httputils"
	"github.com/ViBiOh/httputils/v3/pkg/logger"
	"github.com/ViBiOh/httputils/v3/pkg/owasp"
	"github.com/ViBiOh/httputils/v3/pkg/prometheus"
)

const (
	readingsPath = "/readings"
	tagsPath     = "/tags"

	docPath = "doc/"
)

func main() {
	fs := flag.NewFlagSet("eponae-api", flag.ExitOnError)

	serverConfig := httputils.Flags(fs, "")
	alcotestConfig := alcotest.Flags(fs, "")
	prometheusConfig := prometheus.Flags(fs, "prometheus")
	owaspConfig := owasp.Flags(fs, "")
	corsConfig := cors.Flags(fs, "cors")

	dbConfig := db.Flags(fs, "db")
	authConfig := auth.Flags(fs, "auth")
	basicConfig := basic.Flags(fs, "basic")

	readingsConfig := crud.Flags(fs, "readings")
	tagsConfig := crud.Flags(fs, "tags")

	logger.Fatal(fs.Parse(os.Args[1:]))

	alcotest.DoAndExit(alcotestConfig)

	apiDB, err := db.New(dbConfig)
	logger.Fatal(err)

	tagService := tag.New(apiDB)
	readingTagService := readingtag.New(apiDB, tagService)
	readingService := reading.New(apiDB, readingTagService, tagService)

	readingsApp := crud.New(readingsConfig, readingService)
	tagsApp := crud.New(tagsConfig, tagService)

	readingsHandler := http.StripPrefix(readingsPath, readingsApp.Handler())
	tagsHandler := http.StripPrefix(tagsPath, tagsApp.Handler())

	apihandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, readingsPath) {
			readingsHandler.ServeHTTP(w, r)
			return
		}

		if strings.HasPrefix(r.URL.Path, tagsPath) {
			tagsHandler.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, path.Join(docPath, r.URL.Path))
	})

	server := httputils.New(serverConfig)
	server.Health(httputils.HealthHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if db.Ping(apiDB) {
			w.WriteHeader(http.StatusNoContent)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	})))
	server.Middleware(prometheus.New(prometheusConfig))
	server.Middleware(owasp.New(owaspConfig))
	server.Middleware(cors.New(corsConfig))
	server.Middleware(auth.NewService(authConfig, identService.NewBasic(basicConfig, apiDB)))
	server.ListenServeWait(apihandler)
}
