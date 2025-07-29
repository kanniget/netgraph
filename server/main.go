package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

type Graph struct {
	Nodes []Node `json:"nodes"`
	Links []Link `json:"links"`
}

type Node struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
}

type Link struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/api/graph", graphHandler).Methods("GET")

	// Serve static files from frontend/public
	fs := http.FileServer(http.Dir("frontend/public"))
	r.PathPrefix("/").Handler(fs)

	log.Println("Server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func graphHandler(w http.ResponseWriter, r *http.Request) {
	f, err := os.Open("data/graph.json")
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
