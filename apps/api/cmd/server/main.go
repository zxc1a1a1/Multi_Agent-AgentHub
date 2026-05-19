package main

import (
	"log"
	"net/http"

	"github.com/your-org/multi-agent-framework/apps/api/internal/config"
	"github.com/your-org/multi-agent-framework/apps/api/internal/httpapi"
)

func main() {
	cfg := config.Load()
	router := httpapi.NewRouter(cfg)

	addr := cfg.Host + ":" + cfg.Port
	log.Printf("api server listening on http://%s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}
