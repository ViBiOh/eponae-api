package main

import (
	"flag"
	"log"
	"net/http"
	"strings"

	"github.com/NYTimes/gziphandler"
	"github.com/ViBiOh/alcotest/alcotest"
	"github.com/ViBiOh/charts-api/charts"
	"github.com/ViBiOh/charts-api/healthcheck"
	"github.com/ViBiOh/httputils"
	"github.com/ViBiOh/httputils/cert"
	"github.com/ViBiOh/httputils/cors"
	"github.com/ViBiOh/httputils/db"
	"github.com/ViBiOh/httputils/owasp"
	"github.com/ViBiOh/httputils/prometheus"
	"github.com/ViBiOh/httputils/rate"
)

const healthcheckPath = `/health`

var chartsHandler = charts.Handler()
var healthcheckHandler = http.StripPrefix(healthcheckPath, healthcheck.Handler())
var restHandler http.Handler

func handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, healthcheckPath) {
			healthcheckHandler.ServeHTTP(w, r)
		} else {
			chartsHandler.ServeHTTP(w, r)
		}
	})
}

func main() {
	url := flag.String(`c`, ``, `URL to check`)
	port := flag.String(`port`, `1080`, `Listen port`)
	tls := flag.Bool(`tls`, false, `Serve TLS content`)
	dbConfig := db.Flags(``)
	corsConfig := cors.Flags(``)
	flag.Parse()

	if *url != `` {
		alcotest.Do(url)
		return
	}

	chartsDB, err := db.GetDB(dbConfig)
	if err != nil {
		log.Printf(`Error while initializing database: %v`, err)
	} else {
		log.Print(`Database ready`)
	}

	log.Printf(`Starting server on port %s`, *port)

	if err := healthcheck.Init(chartsDB); err != nil {
		log.Printf(`Error while initializing healthcheck: %v`, err)
	}
	if err := charts.Init(chartsDB); err != nil {
		log.Printf(`Error while initializing charts: %v`, err)
	}

	restHandler = prometheus.Handler(`http`, rate.Handler(gziphandler.GzipHandler(owasp.Handler(cors.Handler(corsConfig, handler())))))
	server := &http.Server{
		Addr:    `:` + *port,
		Handler: restHandler,
	}

	var serveError = make(chan error)
	go func() {
		defer close(serveError)
		if *tls {
			log.Print(`Listening with TLS enabled`)
			serveError <- cert.ListenAndServeTLS(server)
		} else {
			serveError <- server.ListenAndServe()
		}
	}()

	httputils.ServerGracefulClose(server, serveError, nil)
}
