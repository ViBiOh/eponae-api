package conservatories

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/ViBiOh/httputils/db"
	"github.com/ViBiOh/httputils/httperror"
	"github.com/ViBiOh/httputils/httpjson"
	"github.com/ViBiOh/httputils/pagination"
)

const (
	healthcheckPath = `/health`
	defaultPageSize = uint(10)
	maxPageSize     = uint(50)
	defaultSort     = `name`
)

// App stores informations and secret of API
type App struct {
	db *sql.DB
}

// NewApp creates new App from Flags' config
func NewApp(db *sql.DB) *App {
	return &App{
		db: db,
	}
}

func (a *App) listCrud(w http.ResponseWriter, r *http.Request) {
	page, pageSize, sort, asc, err := pagination.ParsePaginationParams(r, defaultPageSize, maxPageSize)
	if err != nil {
		httperror.BadRequest(w, err)
		return
	}

	if sort == `` {
		sort = defaultSort
	}

	if count, list, err := a.findConservatories(page, pageSize, sort, asc, r.URL.Query().Get(`q`)); err != nil {
		httperror.InternalServerError(w, err)
	} else if err := httpjson.ResponsePaginatedJSON(w, http.StatusOK, page, pageSize, count, list, httpjson.IsPretty(r.URL.RawQuery)); err != nil {
		httperror.InternalServerError(w, err)
	}
}

func (a *App) aggregate(w http.ResponseWriter, r *http.Request) {
	if count, err := a.countByDepartment(); err != nil {
		httperror.InternalServerError(w, err)
	} else if err := httpjson.ResponseJSON(w, 200, count, httpjson.IsPretty(r.URL.RawQuery)); err != nil {
		httperror.InternalServerError(w, err)
	}
}

func (a *App) aggregateByDepartment(w http.ResponseWriter, r *http.Request, path string) {
	if count, err := a.countByZipOfDepartment(strings.TrimPrefix(path, `/`)); err != nil {
		httperror.InternalServerError(w, err)
	} else if err := httpjson.ResponseJSON(w, 200, count, httpjson.IsPretty(r.URL.RawQuery)); err != nil {
		httperror.InternalServerError(w, err)
	}
}

// Handler for Conservatories request. Should be use with net/http
func (a *App) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method == http.MethodGet && r.URL.Path == healthcheckPath {
			if db.Ping(a.db) {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
			}
			return
		}

		if r.Method == http.MethodGet && (r.URL.Path == `/` || r.URL.Path == ``) {
			a.listCrud(w, r)
		} else if r.Method == http.MethodGet && r.URL.Path == `/aggregate` {
			a.aggregate(w, r)
		} else if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, `/aggregate/department`) {
			a.aggregateByDepartment(w, r, strings.TrimPrefix(r.URL.Path, `/aggregate/department`))
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}
