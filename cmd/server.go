package main

import (
	"fmt"
	"io"
	log2 "log"
	"math/rand"
	"net/http"
	url2 "net/url"
	"os"
	"strings"

	"github.com/hashicorp/golang-lru/v2"
)

var (
	cache, _ = lru.New[string, string](1024)
	log      = log2.New(os.Stdout, "", log2.Ldate|log2.Ltime)
)

const (
	alphapets = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	BaseURL   = "http://localhost:8080/"
)

func StartServer() {
	addr := fmt.Sprintf(":%d", 8080)
	http.HandleFunc("POST /shorten", Shorten)
	http.HandleFunc("POST /check", Check)
	http.HandleFunc("GET /{key}", Redirect)
	http.HandleFunc("GET /health", Health)

	log.Println("Starting Server at port 8080")
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Println("Server Failed:", err)
		os.Exit(1)
	}
}

func Shorten(writer http.ResponseWriter, req *http.Request) {
	const mxLen = 1024
	reader := http.MaxBytesReader(writer, req.Body, mxLen)
	body, err := io.ReadAll(reader)
	if err != nil {
		http.Error(writer, "URL too long", http.StatusRequestEntityTooLarge)
		return
	}

	origin := string(body)
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
	writer.WriteHeader(http.StatusOK)
	_, err = writer.Write([]byte(newUrl))
	if err != nil {
		log.Println("Failed to write response:", err)
		return
	}
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

func Health(w http.ResponseWriter, r *http.Request) {
	err := pool.Ping(ctx)
	if err != nil {
		log.Println("Database connection failed:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(200)
	if _, err := w.Write([]byte("OK")); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func Check(w http.ResponseWriter, r *http.Request) {
	closer := r.Body
	body, err := io.ReadAll(closer)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	url := string(body)
	key := strings.TrimPrefix(url, BaseURL)
	if len(key) != 6 || key == url {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	origin, found := getAndCache(key)
	if !found {
		http.Error(w, "Target URL Not Found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err = w.Write([]byte(origin)); err != nil {
		log.Println("Failed to write response:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
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
