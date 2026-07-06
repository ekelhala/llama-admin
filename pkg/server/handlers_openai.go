package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"llama-admin/pkg/instance"
	"llama-admin/pkg/validation"
)

type OpenAIListInstancesResponse struct {
	Object string           `json:"object"`
	Data   []OpenAIInstance `json:"data"`
}

type OpenAIInstance struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

type OpenAIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param,omitempty"`
	Code    int    `json:"code,omitempty"`
}

type OpenAIErrorResponse struct {
	Error OpenAIError `json:"error"`
}

func (h *Handler) OpenAIListInstances(w http.ResponseWriter, r *http.Request) {
	instances, err := h.Manager.ListInstances()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	data := make([]OpenAIInstance, 0, len(instances))
	for _, inst := range instances {
		data = append(data, OpenAIInstance{
			ID:      inst.Name,
			Object:  "model",
			Created: inst.CreatedAt,
			OwnedBy: "llama-admin",
		})
	}

	writeJSON(w, http.StatusOK, OpenAIListInstancesResponse{
		Object: "list",
		Data:   data,
	})
}

func (h *Handler) OpenAIProxy(w http.ResponseWriter, r *http.Request) {
	// The model field in the request body is the instance name only.
	// Each instance serves exactly one model, so no splitting or rewriting needed.

	// Read the entire body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "failed to read request body", "invalid_request_error")
		return
	}
	defer r.Body.Close()

	// Require model field
	var body map[string]any
	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &body); err != nil {
			writeOpenAIError(w, http.StatusBadRequest, "invalid JSON: "+err.Error(), "invalid_request_error")
			return
		}
	}

	modelRaw, ok := body["model"]
	if !ok || modelRaw == nil {
		writeOpenAIError(w, http.StatusBadRequest, "model is required", "invalid_request_error")
		return
	}
	model, ok := modelRaw.(string)
	if !ok || model == "" {
		writeOpenAIError(w, http.StatusBadRequest, "model must be a non-empty string", "invalid_request_error")
		return
	}

	// model = instance name
	instanceName := model

	// Validate instance name
	if !validation.IsValidInstanceName(instanceName) {
		writeOpenAIError(w, http.StatusBadRequest, "invalid instance name: "+instanceName, "invalid_request_error")
		return
	}

	// Get instance
	inst, err := h.Manager.GetInstance(instanceName)
	if err != nil {
		writeOpenAIError(w, http.StatusNotFound, fmt.Sprintf("instance %q not found", instanceName), "not_found_error")
		return
	}

	// Check shutting down
	if inst.Status() == instance.StatusShuttingDown {
		writeOpenAIError(w, http.StatusServiceUnavailable, "instance_shutting_down", "unavailable")
		return
	}

	// Only proxy to instances that are already running
	if inst.Status() != instance.StatusRunning {
		writeOpenAIError(w, http.StatusServiceUnavailable, "instance_not_running", "unavailable")
		return
	}

	// Proxy the request body unchanged
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	r.ContentLength = int64(len(bodyBytes))

	inst.Proxy().ServeHTTP(w, r)
}

func writeOpenAIError(w http.ResponseWriter, status int, message, typ string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(OpenAIErrorResponse{
		Error: OpenAIError{
			Message: message,
			Type:    typ,
			Code:    status,
		},
	})
}
