package http

import(
	"enconding/json"
	"net/http"
)

func RootHandler(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

func HealthHandler(w http.ResponseWriter, r *http.Request){
	w.WriteHeader(http.StatusNoContent)
}