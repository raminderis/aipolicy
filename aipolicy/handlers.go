package aipolicy

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// AddHandler handles POST /policies.
func AddHandler(w http.ResponseWriter, r *http.Request) {
	var p Policy
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := createPolicy(&p); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

// GetHandler handles GET /policies/{id}.
func GetHandler(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid UUID")
		return
	}
	p, err := getPolicy(id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "policy not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// UpdateHandler handles PUT /policies/{id}.
// It performs a partial update: only fields provided in the request are updated.
func UpdateHandler(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid UUID")
		return
	}

	// Get existing policy
	existing, err := getPolicy(id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "policy not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Read body once
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "cannot read request body")
		return
	}
	defer r.Body.Close()

	// Check which fields were provided in the JSON
	var updateData map[string]interface{}
	if err := json.Unmarshal(body, &updateData); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Decode into struct
	var update Policy
	if err := json.Unmarshal(body, &update); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Merge: only update fields that were provided in the request
	if _, ok := updateData["policy_id"]; ok {
		existing.PolicyID = update.PolicyID
	}
	if _, ok := updateData["name"]; ok {
		existing.Name = update.Name
	}
	if _, ok := updateData["remote_mcp_service"]; ok {
		existing.RemoteMCPService = update.RemoteMCPService
	}
	if _, ok := updateData["resource_access_request"]; ok {
		existing.ResourceAccessRequest = update.ResourceAccessRequest
	}
	if _, ok := updateData["environment"]; ok {
		existing.Environment = update.Environment
	}
	if _, ok := updateData["enabled"]; ok {
		existing.Enabled = update.Enabled
	}
	if _, ok := updateData["priority"]; ok {
		existing.Priority = update.Priority
	}
	if _, ok := updateData["conditions"]; ok {
		existing.Conditions = update.Conditions
	}
	if _, ok := updateData["description"]; ok {
		existing.Description = update.Description
	}

	// Update in DB
	if err := updatePolicy(id, existing); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "policy not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, existing)
}

// DeleteHandler handles DELETE /policies/{id}.
func DeleteHandler(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid UUID")
		return
	}
	if err := deletePolicy(id); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "policy not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// EvaluateHandler handles POST /decide.
// It finds all enabled policies matching the request, evaluates their
// conditions server-side, and returns { "allowed": true } if at least one
// policy passes (highest-priority first).
func EvaluateHandler(w http.ResponseWriter, r *http.Request) {
	var req DecideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	log.Printf("[DECIDE] Request: service=%s resource=%s environment=%s",
		req.RemoteMCPService, req.ResourceAccessRequest, req.Environment)

	policies, err := findEnabledPolicies(req.RemoteMCPService, req.ResourceAccessRequest, req.Environment)
	if err != nil {
		log.Printf("[DECIDE] ERROR: %v", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Printf("[DECIDE] Found %d matching enabled policies", len(policies))

	allowed := false
	for _, p := range policies {
		log.Printf("[DECIDE] Evaluating policy %s (priority=%d)", p.PolicyID, p.Priority)
		if evaluateConditions(p.Conditions) {
			allowed = true
			log.Printf("[DECIDE] ✓ ACCEPT: policy %s passed conditions", p.PolicyID)
			break
		}
		log.Printf("[DECIDE] ✗ policy %s failed conditions", p.PolicyID)
	}

	if !allowed {
		log.Printf("[DECIDE] ✗ DENY: no matching policies passed conditions")
	}

	writeJSON(w, http.StatusOK, DecideResponse{Allowed: allowed})
}
