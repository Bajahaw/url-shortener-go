package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	log2 "log"
	"math/rand"
	"net/http"
	url2 "net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/hashicorp/golang-lru/v2"
)

var (
	cache, _  = lru.New[string, string](1024)
	log       = log2.New(os.Stdout, "", log2.Ldate|log2.Ltime)
	alphapets = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	BaseURL   = os.Getenv("BASE_URL")
)

func StartServer() {
	addr := fmt.Sprintf(":%d", 8080)

	server := &http.Server{
		Addr:         addr,
		Handler:      CORS(http.DefaultServeMux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	http.HandleFunc("POST /shorten", Shorten)
	http.HandleFunc("POST /check", Check)
	http.HandleFunc("GET /{key}", Redirect)
	http.HandleFunc("GET /health", Health)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Println("Starting Server at port 8080")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server Failed: %v", err)
		}
	}()

	<-stop

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown Failed: %v", err)
	}

	log.Println("Server gracefully stopped")
}

func Shorten(writer http.ResponseWriter, req *http.Request) {
	origin, err := extractBody(writer, req)
	if err != nil {
		log.Println("URL length exceeded:", err)
		http.Error(writer, "URL length exceeded", http.StatusBadRequest)
		return
	}

	if _, err = url2.ParseRequestURI(origin); err != nil {
		msg := fmt.Sprintf("Invalid URL: %v", err)
		http.Error(writer, msg, http.StatusBadRequest)
		return
	}

	key := genKey()
	cache.Add(key, origin)
	if err := SaveURL(key, origin); err != nil {
		log.Println("Failed to save URL:", err)
		http.Error(writer, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	newUrl := BaseURL + key
	httpTextResponse(writer, http.StatusCreated, newUrl)
}

func Redirect(writer http.ResponseWriter, req *http.Request) {
	key := req.PathValue("key")
	if len(key) != 6 {
		fmt.Println("Invalid key format:", key)
		http.Error(writer, "Invalid key", http.StatusBadRequest)
		return
	}

	origin, found := getAndCache(key)
	if !found {
		log.Println("Target URL not found for key:", key)
		http.Error(writer, "Target URL Not Found", http.StatusNotFound)
		return
	}

	http.Redirect(writer, req, origin, http.StatusFound)
}

func Health(w http.ResponseWriter, _ *http.Request) {
	rows, err := pool.Query(ctx, "SELECT 1")
	if err != nil {
		log.Println("Database connection failed:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	httpTextResponse(w, http.StatusOK, "OK")
}

func Check(w http.ResponseWriter, r *http.Request) {
	url, err := extractBody(w, r)
	if err != nil {
		log.Println("Failed to read request url:", err)
		http.Error(w, "Invalid Request Body", http.StatusBadRequest)
		return
	}
	key := strings.TrimPrefix(url, BaseURL)
	if len(key) != 6 || key == url {
		log.Println("Invalid URL host! 3rd party sites are not yet supported", url)
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	origin, found := getAndCache(key)
	if !found {
		http.Error(w, "Target URL Not Found", http.StatusNotFound)
		return
	}

	httpTextResponse(w, http.StatusOK, origin)
}

// CORS middleware
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

//////////////////////////////// HELPER FUNCTIONS ////////////////////////////////

func httpTextResponse(w http.ResponseWriter, status int, message string) {
	log.Println("Sending response:", status, message)
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(status)
	if _, err := w.Write([]byte(message)); err != nil {
		log.Println("Failed to write response:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func extractBody(writer http.ResponseWriter, req *http.Request) (string, error) {
	const mxLen = 2048
	reader := http.MaxBytesReader(writer, req.Body, mxLen)
	defer func(reader io.ReadCloser) {
		err := reader.Close()
		if err != nil {
			log.Println("Failed to close request body:", err)
		}
	}(reader)
	body, err := io.ReadAll(reader)
	return string(body), err
}

func getAndCache(key string) (string, bool) {
	target, found := cache.Get(key)
	if !found {
		t, err := GetURL(key)
		if err != nil {
			return "", false
		}
		target = t
		cache.Add(key, t)
	}
	return target, true
}

func genKey() string {
	var keyArray [6]byte
	for i := 0; i < 6; i++ {
		keyArray[i] = alphapets[rand.Intn(len(alphapets))]
	}
	return string(keyArray[:])
}
