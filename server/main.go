package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
)

type Graph struct {
	Nodes []Node `json:"nodes"`
	Links []Link `json:"links"`
}

type Node struct {
        ID   string                 `json:"id"`
        Type string                 `json:"type"`
        Name string                 `json:"name"`
        Info map[string]interface{} `json:"info,omitempty"`
}

type Link struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/api/graph", graphHandler).Methods("GET")
	r.HandleFunc("/api/files", filesHandler).Methods("GET")

	// Serve static files from frontend/public
	fs := http.FileServer(http.Dir("frontend/public"))
	r.PathPrefix("/").Handler(fs)

	log.Println("Server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func graphHandler(w http.ResponseWriter, r *http.Request) {
	file := r.URL.Query().Get("file")
	if file == "" {
		file = "graph.json"
	}
	file = filepath.Base(file)
	f, err := os.Open(filepath.Join("data", file))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	var g Graph
	if err := json.NewDecoder(f).Decode(&g); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(g)
}

func filesHandler(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir("data")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			files = append(files, e.Name())
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}
