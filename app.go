package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()

	r.Path("/echo").Methods("POST").
		HandlerFunc(echoHandler)

	r.Path("/healthz").Methods("GET").
		HandlerFunc(healthz)

	r.Path("/auth/info/googlejwt").Methods("GET").
		HandlerFunc(authInfoHandler)
	r.Path("/auth/info/googleidtoken").Methods("GET").
		HandlerFunc(authInfoHandler)
	r.Path("/auth/info/firebase").Methods("GET", "OPTIONS").
		Handler(corsHandler(authInfoHandler))
	r.Path("/auth/info/auth0").Methods("GET").
		HandlerFunc(authInfoHandler)

	http.Handle("/", r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

// echoHandler reads a JSON object from the body, and writes it back out.
func echoHandler(w http.ResponseWriter, r *http.Request) {
	var msg interface{}
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		if _, ok := err.(*json.SyntaxError); ok {
			errorf(w, http.StatusBadRequest, "Body was not valid JSON: %v", err)
			return
		}
		errorf(w, http.StatusInternalServerError, "Could not get body: %v", err)
		return
	}

	b, err := json.Marshal(msg)
	if err != nil {
		errorf(w, http.StatusInternalServerError, "Could not marshal JSON: %v", err)
		return
	}
	w.Write(b)
}

// corsHandler wraps a HTTP handler and applies the appropriate responses for Cross-Origin Resource Sharing.
type corsHandler http.HandlerFunc

func (h corsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Headers", "Authorization")
		return
	}
	h(w, r)
}

// authInfoHandler reads authentication info provided by the Endpoints proxy.
func authInfoHandler(w http.ResponseWriter, r *http.Request) {
	encodedInfo := r.Header.Get("X-Endpoint-API-UserInfo")
	if encodedInfo == "" {
		w.Write([]byte(`{"id": "anonymous"}`))
		return
	}

	b, err := base64.StdEncoding.DecodeString(encodedInfo)
	if err != nil {
		errorf(w, http.StatusInternalServerError, "Could not decode auth info: %v", err)
		return
	}
	w.Write(b)
}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte(`{"status": "OK"}`))
}

// errorf writes a swagger-compliant error response.
func errorf(w http.ResponseWriter, code int, format string, a ...interface{}) {
	var out struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}

	out.Code = code
	out.Message = fmt.Sprintf(format, a...)

	b, err := json.Marshal(out)
	if err != nil {
		http.Error(w, `{"code": 500, "message": "Could not format JSON for original message."}`, 500)
		return
	}

	http.Error(w, string(b), code)
}
