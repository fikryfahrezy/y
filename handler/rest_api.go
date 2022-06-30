package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PA-D3RPLA/d3if43-htt-uhomestay/config"
	"github.com/PA-D3RPLA/d3if43-htt-uhomestay/dashboard"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/jackc/pgx/v4/pgxpool"
	httpSwagger "github.com/swaggo/http-swagger"

	mw "github.com/PA-D3RPLA/d3if43-htt-uhomestay/middleware"
)

type RestApiConf struct {
	BuildDate     string
	CommitHash    string
	Conf          config.Config
	PosgrePool    *pgxpool.Pool
	DashboardDeps *dashboard.DashboardDeps
}

func NewRestApi(
	buildDate string,
	commitHash string,
	conf config.Config,
	posgrePool *pgxpool.Pool,
	dashboardDeps *dashboard.DashboardDeps,
) *RestApiConf {
	return &RestApiConf{
		BuildDate:     buildDate,
		CommitHash:    commitHash,
		Conf:          conf,
		PosgrePool:    posgrePool,
		DashboardDeps: dashboardDeps,
	}
}

func (p *RestApiConf) RestApiHandler() {
	trxMidd := mw.NewTrxMiddleware(p.PosgrePool)

	// Basic CORS
	// for more ideas, see: https://developer.github.com/v3/#cross-origin-resource-sharing
	corsMidd := cors.Handler(cors.Options{
		// AllowedOrigins:   []string{"https://foo.com"}, // Use this to allow specific origin hosts
		AllowedOrigins: []string{"https://*", "http://*"},
		// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	})

	// Enable httprate request limiter of 100 requests per minute.
	//
	// In the code example below, rate-limiting is bound to the request IP address
	// via the LimitByIP middleware handler.
	//
	// To have a single rate-limiter for all requests, use httprate.LimitAll(..).
	//
	// Please see _example/main.go for other more, or read the library code.
	rateLMidd := httprate.LimitByIP(100, 1*time.Minute)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(rateLMidd)
	r.Use(corsMidd)

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte("route does not exist"))
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(405)
		w.Write([]byte("method is not valid"))
	})
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	})
	r.Get("/info", func(w http.ResponseWriter, r *http.Request) {
		info := "Built: " + p.BuildDate + ", Commit: " + p.CommitHash
		w.Write([]byte(info))
	})

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/docs/swagger.yaml"), // The url pointing to API definition
	))

	r.Get("/registerform", p.DashboardDeps.RegisterForm)
	r.Get("/positionform", p.DashboardDeps.PositionForm)
	r.Get("/periodform", p.DashboardDeps.PeriodForm)
	r.Post("/positions", p.DashboardDeps.PostPosition)
	r.With(trxMidd).Post("/periods", p.DashboardDeps.PostPeriod)
	r.With(trxMidd).Post("/members", p.DashboardDeps.PostMember)

	r.With(trxMidd).Post("/api/v1/register", p.DashboardDeps.PostRegisterMember)
	r.Post("/api/v1/login/members", p.DashboardDeps.PostLoginMember)
	r.Post("/api/v1/login/admins", p.DashboardDeps.PostLoginAdmin)

	r.Get("/api/v1/members", p.DashboardDeps.GetMembers)
	r.Get("/api/v1/members/{id}", p.DashboardDeps.GetMember)
	r.Get("/api/v1/profile", p.DashboardDeps.GetProfileMember)
	r.With(trxMidd).Post("/api/v1/members", p.DashboardDeps.PostMember)
	r.With(trxMidd).Put("/api/v1/members", p.DashboardDeps.PutMemberProfile)
	r.With(trxMidd).Put("/api/v1/members/{id}", p.DashboardDeps.PutMember)
	r.With(trxMidd).Delete("/api/v1/members/{id}", p.DashboardDeps.DeleteMember)
	r.With(trxMidd).Patch("/api/v1/members/{id}", p.DashboardDeps.PatchMemberApproval)

	r.Get("/api/v1/periods", p.DashboardDeps.GetPeriods)
	r.Get("/api/v1/periods/active", p.DashboardDeps.GetActivePeriod)
	r.Get("/api/v1/periods/{id}/structures", p.DashboardDeps.GetPeriodStructure)
	r.With(trxMidd).Post("/api/v1/periods", p.DashboardDeps.PostPeriod)
	r.Post("/api/v1/periods/goals", p.DashboardDeps.PostGoal)
	r.With(trxMidd).Put("/api/v1/periods/{id}", p.DashboardDeps.PutPeriod)
	r.With(trxMidd).Delete("/api/v1/periods/{id}", p.DashboardDeps.DeletePeriod)
	r.With(trxMidd).Patch("/api/v1/periods/{id}/status", p.DashboardDeps.PatchPeriodStatus)
	r.Get("/api/v1/periods/{id}/goal", p.DashboardDeps.GetOrgPeriodGoal)

	r.Get("/api/v1/positions", p.DashboardDeps.GetPositions)
	r.Get("/api/v1/positions/levels", p.DashboardDeps.GetPositionLevels)
	r.Post("/api/v1/positions", p.DashboardDeps.PostPosition)
	r.With(trxMidd).Put("/api/v1/positions/{id}", p.DashboardDeps.PutPositions)
	r.With(trxMidd).Delete("/api/v1/positions/{id}", p.DashboardDeps.DeletePosition)

	r.Get("/api/v1/documents", p.DashboardDeps.GetDocuments)
	r.Post("/api/v1/documents/dir", p.DashboardDeps.PostDirDocument)
	r.Post("/api/v1/documents/file", p.DashboardDeps.PostFileDocument)
	r.With(trxMidd).Put("/api/v1/documents/dir/{id}", p.DashboardDeps.PutDirDocument)
	r.With(trxMidd).Put("/api/v1/documents/file/{id}", p.DashboardDeps.PutFileDocument)
	r.Get("/api/v1/documents/{id}", p.DashboardDeps.GetDocumentChildren)
	r.With(trxMidd).Delete("/api/v1/documents/{id}", p.DashboardDeps.DeleteDocument)

	r.Post("/api/v1/histories", p.DashboardDeps.PostHistory)
	r.Get("/api/v1/histories", p.DashboardDeps.GetHistory)

	r.Get("/api/v1/blogs", p.DashboardDeps.GetBlogs)
	r.Get("/api/v1/blogs/{id}", p.DashboardDeps.GetBlog)
	r.Post("/api/v1/blogs", p.DashboardDeps.PostBlog)
	r.Put("/api/v1/blogs/{id}", p.DashboardDeps.PutBlogs)
	r.Delete("/api/v1/blogs/{id}", p.DashboardDeps.DeleteBlog)
	r.Post("/api/v1/blogs/image", p.DashboardDeps.PostImage)

	r.Get("/api/v1/cashflows", p.DashboardDeps.GetCashflows)
	r.Post("/api/v1/cashflows", p.DashboardDeps.PostCashflow)
	r.Put("/api/v1/cashflows/{id}", p.DashboardDeps.PutCashflow)
	r.Delete("/api/v1/cashflows/{id}", p.DashboardDeps.DeleteCashflow)

	r.Put("/api/v1/dues/members/monthly/{id}", p.DashboardDeps.PutMemberDues)
	r.Patch("/api/v1/dues/members/monthly/{id}", p.DashboardDeps.PatchMemberDues)
	r.Post("/api/v1/dues/members/monthly/{id}", p.DashboardDeps.PostMemberDues)
	r.Get("/api/v1/dues/members/{id}", p.DashboardDeps.GetMemberDues)
	r.Get("/api/v1/dues/{id}/members", p.DashboardDeps.GetMembersDues)

	r.Get("/api/v1/dues", p.DashboardDeps.GetDues)
	r.Post("/api/v1/dues", p.DashboardDeps.PostDues)
	r.Get("/api/v1/dues/{id}/check", p.DashboardDeps.GetPaidDues)
	r.Put("/api/v1/dues/{id}", p.DashboardDeps.PutDues)
	r.Delete("/api/v1/dues/{id}", p.DashboardDeps.DeleteDues)

	r.Get("/api/v1/dashboard", p.DashboardDeps.GetPublicDashboard)
	r.Get("/api/v1/dashboard/private", p.DashboardDeps.GetPrivateDashboard)

	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, "docs"))
	ChiFileServer(r, "/docs", filesDir)

	http.ListenAndServe(fmt.Sprintf(":%s", p.Conf.Port), r)
}

// Ref:
// https://github.com/go-chi/chi/blob/master/_examples/fileserver/main.go
//
// FileServer conveniently sets up a http.FileServer handler to serve
// static files from a http.FileSystem.
func ChiFileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}
