package main

import (
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"runtime/debug"
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

var (
	//go:embed services.yaml
	servicesByte []byte
	//go:embed app.html
	appHTML []byte
	//go:embed assets/*
	assets embed.FS

	appTemplate *template.Template

	json = jsoniter.ConfigCompatibleWithStandardLibrary

	voidInfo types.VoidInfo
)

// Load the services.yaml file into the state here because we use go:embed
func init() {
	state.Init()

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

	voidInfo = types.VoidInfo{
		Version: "1",
	}

	var commit string

	// Use runtime/debug vcs.revision to get the git commit hash
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				commit = setting.Value
			}
		}
	}

	if commit == "" {
		commit = "unknown"
	}

	voidInfo.Commit = commit
}

func dataHandlerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// limit body to 10mb
		r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)

		var allowedHeaders = []string{"Content-Type", "Authorization"}

		reqHeaderList := strings.Split(r.Header.Get("Access-Control-Request-Headers"), ",")

		for _, name := range reqHeaderList {
			if name == "" {
				continue
			}

			state.Logger.Info(name)

			if strings.HasPrefix(strings.ToLower(name), "x-") {
				allowedHeaders = append(allowedHeaders, strings.ReplaceAll(name, " ", ""))
			}
		}

		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(allowedHeaders, ", "))
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

		if r.URL.Query().Get("data") != "" {
			split := strings.Split(r.URL.Query().Get("data"), "|")

			if len(split) > 0 {
				r.Method = split[0]
			}

			if len(split) > 1 {
				// All the remaining sections are considered PATH
				r.URL.Path = strings.Join(split[1:], "/")
			}
		}

		// Handle options immediately here
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
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
		dataHandlerMiddleware,
		middleware.RealIP,
		middleware.CleanPath,
		zapchi.Logger(state.Logger, "api"),
		middleware.Timeout(30*time.Second),
	)

	subbedAssets, err := fs.Sub(assets, "assets")

	if err != nil {
		panic(err)
	}

	r.HandleFunc("/__voidStatic/*", func(w http.ResponseWriter, r *http.Request) {
		http.StripPrefix("/__voidStatic", http.FileServer(http.FS(subbedAssets))).ServeHTTP(w, r)
	})

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

		if slices.Contains(state.Services.APIUrls, hostname) {
			w.WriteHeader(http.StatusRequestTimeout)
			w.Header().Set("Content-Type", "application/json")

			apiCtx := types.APICtx{
				Message: "This service is down for maintenance...",
				Service: service,
				Info:    voidInfo,
			}

			json.NewEncoder(w).Encode(apiCtx)
			return
		}

		htmlCtx := types.HTMLCtx{
			MatchedService: service,
			Path:           r.URL.Path,
			Hostname:       hostname,
			Info:           voidInfo,
			Redirect:       r.URL.Query().Get("src"),
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

	err = http.ListenAndServe(":1292", r)

	if err != nil {
		panic(err)
	}
}
