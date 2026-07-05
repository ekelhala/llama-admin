package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"llama-admin/pkg/backends"
	"llama-admin/pkg/instance"
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
	// Read the entire body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	defer r.Body.Close()

	// Parse JSON
	var body map[string]any
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	// Require model field
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

	// Split instance/model
	instanceName, modelName := backends.SplitInstanceModel(model)

	// Validate instance name
	if err := backends.ValidateInstanceName(instanceName); err != nil {
		writeOpenAIError(w, http.StatusBadRequest, err.Error(), "invalid_request_error")
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
		writeOpenAIError(w, http.StatusServiceUnavailable, "instance_shutting_down", " unavailable")
		return
	}

	// Check if running, start if on-demand
	if inst.Status() != instance.StatusRunning {
		if inst.Opts != nil && inst.Opts.BackendOptions != nil {
			if onDemand, ok := inst.Opts.BackendOptions["on_demand"]; ok {
				if b, ok := onDemand.(bool); ok && b {
					if _, err := h.Manager.StartInstance(instanceName); err != nil {
						writeOpenAIError(w, http.StatusInternalServerError, fmt.Sprintf("failed to start instance: %v", err), "internal_error")
						return
					}
				} else {
					writeOpenAIError(w, http.StatusServiceUnavailable, "instance_not_running", "unavailable")
					return
				}
			} else {
				writeOpenAIError(w, http.StatusServiceUnavailable, "instance_not_running", "unavailable")
				return
			}
		} else {
			writeOpenAIError(w, http.StatusServiceUnavailable, "instance_not_running", "unavailable")
			return
		}
	}

	// Resolve inner model
	if modelName == "" {
		// No "/" in model - set to instance's configured model
		if model, ok := inst.Opts.BackendOptions["model"]; ok {
			if m, ok := model.(string); ok && m != "" {
				modelName = m
			} else {
				modelName = inst.Name
			}
		} else {
			modelName = inst.Name
		}
	}
	body["model"] = modelName

	// Re-marshal body
	newBody, err := json.Marshal(body)
	if err != nil {
		writeOpenAIError(w, http.StatusInternalServerError, "failed to marshal request", "internal_error")
		return
	}

	r.Body = io.NopCloser(strings.NewReader(string(newBody)))
	r.ContentLength = int64(len(newBody))

	// Proxy to instance
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
