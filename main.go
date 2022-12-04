package main

import (
	_ "embed"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"
	"void/state"
	"void/types"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
	"github.com/infinitybotlist/eureka/zapchi"
	jsoniter "github.com/json-iterator/go"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

var (
	//go:embed services.yaml
	servicesByte []byte
	//go:embed app.html
	appHTML []byte

	appTemplate *template.Template
)

// Load the services.yaml file into the state here because we use go:embed
func init() {
	// Parse yaml
	err := yaml.Unmarshal(servicesByte, &state.Services)

	if err != nil {
		log.Fatal("Could not parse services.yaml:", err)
	}

	// Validate the yaml
	validator := validator.New()

	err = validator.Struct(state.Services)

	if err != nil {
		log.Fatal("Could not validate services.yaml:", err)
	}

	appTemplate = template.Must(template.New("app").Parse(string(appHTML)))

	state.Logger.Info("Got services:", state.Services)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// limit body to 10mb
		r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)

		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")

		if r.Method == "OPTIONS" {
			w.Write([]byte{})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	r := chi.NewRouter()

	// A good base middleware stack
	r.Use(
		middleware.Recoverer,
		middleware.RealIP,
		middleware.CleanPath,
		corsMiddleware,
		zapchi.Logger(state.Logger, "api"),
		middleware.Timeout(30*time.Second),
	)

	r.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		// Get the hostname from the request
		hostname := r.Host

		// Also get the root domain from the request
		var rootDomain string

		rootDomainLst := strings.Split(hostname, ".")

		// Then slice last 2 elements IF the length is greater than 2
		if len(rootDomainLst) > 2 {
			rootDomain = strings.Join(rootDomainLst[len(rootDomainLst)-2:], ".")
		} else {
			rootDomain = hostname
		}

		// Find the right service from config
		var service = types.Service{
			Name:    "Unknown Service",
			Domain:  "infinitybots.gg",
			Support: "https://discord.gg/cRuprw9CGz",
			Status:  "https://status.botlist.site/",
		}

		for _, s := range state.Services.Services {
			if s.Domain == rootDomain {
				service = s
				break
			}
		}

		if strings.HasPrefix(hostname, "api.") || slices.Contains(state.Services.APIUrls, hostname) {
			w.WriteHeader(http.StatusRequestTimeout)
			w.Header().Set("Content-Type", "application/json")

			apiCtx := types.APICtx{
				Message: "This service is down for maintenance...",
				Service: service,
			}

			json.NewEncoder(w).Encode(apiCtx)
			return
		}

		htmlCtx := types.HTMLCtx{
			MatchedService: service,
			Path:           r.URL.Path,
			Hostname:       hostname,
		}

		// Set status code of 408
		w.WriteHeader(http.StatusRequestTimeout)
		w.Header().Add("Content-Type", "text/html")

		// Execute the template
		err := appTemplate.Execute(w, htmlCtx)

		if err != nil {
			state.Logger.Error("Could not execute template:", err)
			w.Write([]byte(err.Error()))
		}
	})

	http.ListenAndServe(":3838", r)
}
