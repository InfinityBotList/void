package main

import (
	_ "embed"
	"log"
	"net/http"
	"time"
	"void/state"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
	"github.com/infinitybotlist/eureka/zapchi"
	"gopkg.in/yaml.v3"
)

//go:embed services.yaml
var servicesByte []byte

//go:embed app.html
var appHTML []byte

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
		w.Header().Add("Content-Type", "text/html")
		w.Write(appHTML)
	})

	http.ListenAndServe(":3838", r)
}
