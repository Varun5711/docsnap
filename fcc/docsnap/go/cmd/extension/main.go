package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/docsnap/docsnap/fcc/docsnap/internal/extension"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "stdio" {
		var req extension.CertifyRequest
		if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
			log.Fatal(err)
		}
		result, err := extension.Certify(req)
		if err != nil {
			log.Fatal(err)
		}
		if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
			log.Fatal(err)
		}
		return
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	http.HandleFunc("/certify", func(w http.ResponseWriter, r *http.Request) {
		var req extension.CertifyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		result, err := extension.Certify(req)
		if err != nil {
			http.Error(w, "certification failed", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(result)
	})

	addr := os.Getenv("DOCSNAP_FCC_ADDR")
	if addr == "" {
		addr = ":8787"
	}
	log.Printf("docsnap fcc extension listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
