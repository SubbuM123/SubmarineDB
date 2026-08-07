package main

import (
	"encoding/json"
	"net/http"
)

type kvValuePayload struct {
	Data     string   `json:"data"`
	ParentVideo string `json:"parent_video"`
	Metadata []string `json:"metadata"`
}

type kvRequest struct {
	Key   string         `json:"key"`
	Value kvValuePayload `json:"value"`
}

func (n *Node) put_chunk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req kvRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}

	if req.Key == "" {
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}

	descriptors := make(map[string]struct{}, len(req.Value.Metadata))
	for _, metadata := range req.Value.Metadata {
		descriptors[metadata] = struct{}{}
	}

	if !n.Put(req.Key, Value{Data: req.Value.Data, ParentVideo: req.Value.ParentVideo, Descriptors: descriptors}) {
		http.Error(w, "failed to store key/value pair", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true}`))
}

func (n *Node) get_chunk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}

	ok, value := n.Get(key)
	if !ok {
		http.Error(w, "key not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(value.Data)
}

func (n *Node) delete_chunk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}

	if !n.Delete(key) {
		http.Error(w, "key not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
}
