package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/api/v1/languages", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"languages": []map[string]string{
				{"code": "en", "name": "English"},
				{"code": "ru", "name": "Russian"},
				{"code": "zh", "name": "Chinese"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	http.HandleFunc("/api/v1/translate", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]string
		json.NewDecoder(r.Body).Decode(&req)

		response := map[string]interface{}{
			"translation": "привет",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	log.Println("Mock Service2 running on :8083")
	log.Fatal(http.ListenAndServe(":8083", nil))
}
