package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"io"
	log2 "log"
	"math/rand"
	"net/http"
	url2 "net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hashicorp/golang-lru/v2"
)

var (
	Repo      = NewRepository()
	cache, _  = lru.New[string, string](1024)
	log       = log2.New(os.Stdout, "", log2.Ldate|log2.Ltime)
	alphapets = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	BaseURL   = os.Getenv("BASE_URL")
)

func StartServer() {
	server := &http.Server{
		Addr:         ":8080",
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
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server Failed: %v", err)
		}
	}()

	log.Println("Server started on port 8080")

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
		log.Println(msg)
		http.Error(writer, msg, http.StatusBadRequest)
		return
	}

	key := genKey()
	cache.Add(key, origin)
	if err := Repo.SaveURL(key, origin); err != nil {
		log.Println("Failed to save URL:", err)
		http.Error(writer, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	newUrl := BaseURL + key
	httpTextResponse(writer, http.StatusOK, newUrl)
}

func Redirect(writer http.ResponseWriter, req *http.Request) {
	key := req.PathValue("key")

	origin, found := getAndCache(key)
	if !found || origin == "" {
		log.Println("Target URL not found for key:", key)
		http.NotFound(writer, req)
		return
	}

	http.Redirect(writer, req, origin, http.StatusFound)
}

func Health(w http.ResponseWriter, _ *http.Request) {
	_, err := Repo.GetURL("tst123")
	if err != nil {
		log.Println("Database connection failed:", err)
		http.Error(w, "Health check failed!", http.StatusServiceUnavailable)
		return
	}
	httpTextResponse(w, http.StatusOK, "OK")
}

func Check(w http.ResponseWriter, r *http.Request) {
	url, err := extractBody(w, r)
	if err != nil {
		msg := "Invalid Request Body: " + err.Error()
		log.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	location, err := checkForeignURL(url)
	if err != nil {
		msg := "Failed to check origin URL: " + err.Error()
		log.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	httpTextResponse(w, http.StatusOK, location)
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

//////////////////////////////////////////////////////////////////////////////////
//////////////////////////////// HELPER FUNCTIONS ////////////////////////////////
//////////////////////////////////////////////////////////////////////////////////

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

func checkForeignURL(url string) (string, error) {
	if _, err := url2.ParseRequestURI(url); err != nil {
		return "", err
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}

	location := resp.Header.Get("Location")
	if location == "" {
		location = url
	}

	return location, nil
}

func getAndCache(key string) (string, bool) {
	target, found := cache.Get(key)
	if !found {
		t, err := Repo.GetURL(key)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				cache.Add(key, "")
			}
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
