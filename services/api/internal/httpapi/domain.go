package httpapi

import (
	"log"
	"net/http"
)

func (s Server) domainTrust(w http.ResponseWriter, r *http.Request) {
	trust, err := s.store.DomainTrust(r.Context(), r.PathValue("domain"))
	if err != nil {
		log.Printf("domainTrust: query failed: %v", err)
		writeError(w, http.StatusInternalServerError, "couldn't compute domain trust")
		return
	}
	writeJSON(w, http.StatusOK, trust)
}
