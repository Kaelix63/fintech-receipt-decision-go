package main

import (
	"encoding/json"
	"log"
	"net/http"

	"receipt-fintech-go"
)

func main() {
	client, err := receipt.NewEmailClient()
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/receipts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		var order receipt.Order
		if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		decision, err := receipt.ProcessOrder(r.Context(), client, order)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(decision)
	})
	log.Println("receipt service listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
