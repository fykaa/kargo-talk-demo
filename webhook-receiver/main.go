package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

const (
	port         = "8080"
	webhookPath  = "/webhook"
	secretHeader = "X-Webhook-Secret"
	expectedSecret = "my-super-secret-123"
)

type DockerHubPush struct {
	PushData struct {
		PushedAt string `json:"pushed_at"`
		Tag      string `json:"tag"`
	} `json:"push_data"`
	Repository struct {
		Name     string `json:"name"`
		RepoName string `json:"repo_name"`
	} `json:"repository"`
	CallbackURL string `json:"callback_url"`
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if secret := r.Header.Get(secretHeader); secret != "" {
		if secret != expectedSecret {
			log.Printf("Invalid secret: %s", secret)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	} else {
		http.Error(w, "Missing secret header", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	log.Printf("Webhook received at: %s", r.Header.Get("Date"))
	log.Printf("Headers: %v", r.Header)
	log.Printf("Raw body: %s", string(body))

	var prettyJSON map[string]interface{}
	if json.Unmarshal(body, &prettyJSON) == nil {
		prettyBytes, _ := json.MarshalIndent(prettyJSON, "", "  ")
		log.Printf("Pretty payload:\n%s", string(prettyBytes))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Webhook received successfully",
	})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	http.HandleFunc(webhookPath, webhookHandler)
	http.HandleFunc("/health", healthHandler)

	log.Printf("Starting webhook receiver on port %s", port)
	log.Printf("Webhook endpoint: POST %s", webhookPath)
	log.Printf("Health endpoint: GET /health")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
