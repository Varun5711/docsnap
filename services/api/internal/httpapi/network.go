package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/docsnap/docsnap/services/api/internal/auth"
	"github.com/docsnap/docsnap/services/api/internal/model"
	"github.com/docsnap/docsnap/services/api/internal/store"
)

type signupRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
}

type authResponse struct {
	Token string     `json:"token"`
	User  model.User `json:"user"`
}

func (s Server) signup(w http.ResponseWriter, r *http.Request) {
	var req signupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	email := auth.NormalizeEmail(req.Email)
	if err := auth.ValidateEmail(email); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := auth.ValidatePassword(req.Password); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, _, err := s.store.GetUserByEmail(r.Context(), email); err == nil {
		writeError(w, http.StatusConflict, auth.ErrEmailTaken.Error())
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "signup failed")
		return
	}
	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		displayName = strings.SplitN(email, "@", 2)[0]
	}
	user := model.User{ID: "usr_" + randomHex(10), Email: email, DisplayName: displayName, CreatedAt: time.Now().UTC()}
	if err := s.store.CreateUser(r.Context(), user, hash); err != nil {
		log.Printf("signup: save failed: %v", err)
		writeError(w, http.StatusInternalServerError, "signup failed")
		return
	}

	s.respondWithSession(w, r, user)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s Server) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	email := auth.NormalizeEmail(req.Email)
	user, hash, err := s.store.GetUserByEmail(r.Context(), email)
	if errors.Is(err, store.ErrNotFound) || !auth.CheckPassword(hash, req.Password) {
		writeError(w, http.StatusUnauthorized, auth.ErrInvalidCreds.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "login failed")
		return
	}
	s.respondWithSession(w, r, user)
}

func (s Server) respondWithSession(w http.ResponseWriter, r *http.Request, user model.User) {
	token := auth.NewSessionToken()
	if err := s.store.CreateSession(r.Context(), token, user.ID, time.Now().UTC().Add(auth.SessionDuration)); err != nil {
		log.Printf("session: create failed: %v", err)
		writeError(w, http.StatusInternalServerError, "login failed")
		return
	}
	writeJSON(w, http.StatusOK, authResponse{Token: token, User: user})
}

func (s Server) logout(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if token != "" {
		_ = s.store.DeleteSession(r.Context(), token)
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s Server) me(w http.ResponseWriter, r *http.Request) {
	_, viewerID := s.resolveViewer(r)
	if viewerID == "" {
		writeError(w, http.StatusUnauthorized, "not logged in")
		return
	}
	user, err := s.store.GetUserByID(r.Context(), viewerID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "not logged in")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (s Server) requireUser(w http.ResponseWriter, r *http.Request) (model.User, bool) {
	_, viewerID := s.resolveViewer(r)
	if viewerID == "" {
		writeError(w, http.StatusUnauthorized, "log in to do that")
		return model.User{}, false
	}
	user, err := s.store.GetUserByID(r.Context(), viewerID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "log in to do that")
		return model.User{}, false
	}
	return user, true
}

func (s Server) discover(w http.ResponseWriter, r *http.Request) {
	recent, trending, err := s.store.Discover(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "discover failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"recent": recent, "trending": trending})
}

func (s Server) searchCanonicalClaims(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		writeJSON(w, http.StatusOK, []model.CanonicalClaim{})
		return
	}
	results, err := s.store.FindSimilarCanonicalClaims(r.Context(), query, 20)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "search failed")
		return
	}
	writeJSON(w, http.StatusOK, filterPublicRollups(results))
}

func (s Server) similarClaims(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		writeJSON(w, http.StatusOK, []model.CanonicalClaim{})
		return
	}
	results, err := s.store.FindSimilarCanonicalClaims(r.Context(), query, 5)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "search failed")
		return
	}
	writeJSON(w, http.StatusOK, filterPublicRollups(results))
}

func filterPublicRollups(rollups []model.CanonicalClaim) []model.CanonicalClaim {
	for i := range rollups {
		public := make([]model.Claim, 0, len(rollups[i].Claims))
		for _, c := range rollups[i].Claims {
			if c.Visibility == "public" {
				public = append(public, c)
			}
		}
		rollups[i].Claims = public
	}
	return rollups
}

var slugPattern = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(text string, suffix string) string {
	slug := strings.ToLower(strings.TrimSpace(text))
	slug = slugPattern.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > 60 {
		slug = strings.Trim(slug[:60], "-")
	}
	if slug == "" {
		slug = "claim"
	}
	return slug + "-" + suffix
}

func (s Server) getCanonicalClaim(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	cc, err := s.store.GetCanonicalClaimBySlug(r.Context(), slug)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "claim not found")
		return
	}
	if err != nil {
		log.Printf("getCanonicalClaim: read failed: %v", err)
		writeError(w, http.StatusInternalServerError, "read failed")
		return
	}

	fullAccess, viewerID := s.resolveViewer(r)
	visible := make([]model.Claim, 0, len(cc.Claims))
	for _, c := range cc.Claims {
		if canView(c, fullAccess, viewerID) {
			visible = append(visible, c)
		}
	}
	cc.Claims = visible
	writeJSON(w, http.StatusOK, cc)
}

type evidenceContributionRequest struct {
	Type string `json:"type"`
	URL  string `json:"url"`
	Note string `json:"note"`
}

var validContributionTypes = map[string]bool{"support": true, "contradict": true, "context": true, "correction": true}

func (s Server) addEvidence(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var req evidenceContributionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Type = strings.ToLower(strings.TrimSpace(req.Type))
	if !validContributionTypes[req.Type] {
		writeError(w, http.StatusBadRequest, "type must be support, contradict, context, or correction")
		return
	}
	if strings.TrimSpace(req.URL) == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}

	claimID := r.PathValue("id")
	if _, err := s.store.GetClaim(r.Context(), claimID); errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "claim not found")
		return
	}

	contribution := model.EvidenceContribution{
		ID:            "con_" + randomHex(10),
		ClaimID:       claimID,
		ContributorID: user.ID,
		Type:          req.Type,
		URL:           strings.TrimSpace(req.URL),
		Note:          strings.TrimSpace(req.Note),
		CreatedAt:     time.Now().UTC(),
	}
	if err := s.store.AddEvidenceContribution(r.Context(), contribution); err != nil {
		log.Printf("addEvidence: save failed: %v", err)
		writeError(w, http.StatusInternalServerError, "save failed")
		return
	}

	updated, err := s.store.GetClaim(r.Context(), claimID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "read failed")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s Server) forkClaim(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	parentID := r.PathValue("id")
	newID := "cl_" + randomHex(10)
	forked, err := s.store.ForkClaim(r.Context(), parentID, newID, user.ID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "claim not found")
		return
	}
	if err != nil {
		log.Printf("forkClaim: failed: %v", err)
		writeError(w, http.StatusInternalServerError, "fork failed")
		return
	}
	writeJSON(w, http.StatusCreated, forked)
}

type publishRequest struct {
	Visibility string `json:"visibility"`
}

var validVisibilities = map[string]bool{"private": true, "unlisted": true, "public": true}

func (s Server) publishClaim(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var req publishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Visibility = strings.ToLower(strings.TrimSpace(req.Visibility))
	if !validVisibilities[req.Visibility] {
		writeError(w, http.StatusBadRequest, "visibility must be private, unlisted, or public")
		return
	}

	claimID := r.PathValue("id")
	claim, err := s.store.GetClaim(r.Context(), claimID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "claim not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "read failed")
		return
	}

	if claim.PublishedBy != "" && claim.PublishedBy != user.ID {
		writeError(w, http.StatusForbidden, "only the owner can change this claim's visibility")
		return
	}

	canonicalID := claim.CanonicalClaimID
	if canonicalID == "" {

		matches, err := s.store.FindSimilarCanonicalClaims(r.Context(), claim.Text, 1)
		if err == nil && len(matches) > 0 {
			canonicalID = matches[0].ID
		} else {
			cc := model.CanonicalClaim{ID: "cc_" + randomHex(10), Text: claim.Text, CreatedAt: time.Now().UTC()}
			cc.Slug = slugify(claim.Text, randomHex(4))
			if err := s.store.CreateCanonicalClaim(r.Context(), cc); err != nil {
				log.Printf("publishClaim: create canonical claim failed: %v", err)
				writeError(w, http.StatusInternalServerError, "publish failed")
				return
			}
			canonicalID = cc.ID
		}
	}

	if err := s.store.PublishClaim(r.Context(), claimID, canonicalID, req.Visibility, user.ID); err != nil {
		log.Printf("publishClaim: failed: %v", err)
		writeError(w, http.StatusInternalServerError, "publish failed")
		return
	}

	updated, err := s.store.GetClaim(r.Context(), claimID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "read failed")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}
