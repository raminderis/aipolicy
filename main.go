package main

import (
	"log"
	"net/http"
	"os"

	"aipolicy/aipolicy"

	"github.com/go-chi/chi"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("warning: could not load .env file: %v", err)
	}

	port := getEnv("POLICYMGR_LISTENING_PORT", "8081")
	addr := ":" + port

	router := chi.NewRouter()
	router.Post("/policies", aipolicy.AddHandler)
	router.Put("/policies/{id}", aipolicy.UpdateHandler)
	router.Delete("/policies/{id}", aipolicy.DeleteHandler)
	router.Post("/decide", aipolicy.EvaluateHandler)
	router.Get("/policies/{id}", aipolicy.GetHandler)

	log.Printf("policy mgr service listening on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}
