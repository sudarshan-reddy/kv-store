package kv

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type Server struct {
	mux  *http.ServeMux
	db   Store
	addr string
}

func NewHTTPServer(store Store, addr string) *Server {
	mux := http.NewServeMux()
	server := &Server{db: store, addr: addr, mux: mux}
	mux.HandleFunc("/set", server.setHandler)
	mux.HandleFunc("/get", server.getHandler)
	mux.HandleFunc("/updateBulk", server.updateBulkHandler)
	mux.HandleFunc("/delete", server.deleteHandler)
	return server
}

func (s *Server) Start() error {
	log.Printf("Server running at: %s\n", s.addr)
	// TODO: Consider implementing TLS support with http.ListenAndServeTLS
	return http.ListenAndServe(s.addr, s.mux)
}

type Response struct {
	Value interface{} `json:"value"`
}

func (s *Server) getHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	value, err := s.db.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	resp := Response{Value: value}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) setHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var kv Pair
	err := json.NewDecoder(r.Body).Decode(&kv)
	r.Body.Close()
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if err := s.db.Put(kv.Key, kv.Value); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (s *Server) updateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var kv Pair

	err := json.NewDecoder(r.Body).Decode(&kv)
	r.Body.Close()
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if err := s.db.Update(kv.Key, kv.Value); err != nil {
		if err, ok := err.(*notFoundError); ok && err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

}

// TODO: This whole thing is memory/allocation intensive. I tend to keep the
// body in memory and later do the same inside the data store with storing the
// rollback information and what not.
// Maybe I should think of a streaming solution here eventually.
func (s *Server) updateBulkHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var kvs []Pair
	err := json.NewDecoder(r.Body).Decode(&kvs)
	r.Body.Close()
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	updatedKeys, err := s.db.BatchUpdate(ctx, kvs)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if len(updatedKeys) != len(kvs) {
		http.Error(w, "Partial update", http.StatusPartialContent)
		return
	} else {
		w.WriteHeader(http.StatusOK)
	}

	resp := Response{Value: updatedKeys}
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) deleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	key := r.URL.Query().Get("key")
	if err := s.db.Delete(key); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
