package main

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	url2 "net/url"
	"regexp"
)

var (
	cache = make(map[string]string, 1000)
)

const (
	alphapets = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	BaseURL   = "http://localhost:8080/"
)

func StartServer() {

	addr := fmt.Sprintf(":%d", 8080)
	http.HandleFunc("GET /{key}", Redirect)
	http.HandleFunc("POST /shorten", Shorten)

	err := http.ListenAndServe(addr, http.DefaultServeMux)
	if err != nil {
		return
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

	var keyArray [6]byte
	for i := 0; i < 6; i++ {
		keyArray[i] = alphapets[rand.Intn(len(alphapets))]
	}
	key := string(keyArray[:])

	cache[key] = origin
	newUrl := BaseURL + key
	writer.WriteHeader(http.StatusOK)
	_, err = writer.Write([]byte(newUrl))
	if err != nil {
		return
	}
}

func Redirect(writer http.ResponseWriter, req *http.Request) {
	key := req.PathValue("key")
	if valid, _ := regexp.MatchString("[a-zA-Z]{6}", key); !valid {
		fmt.Println("Invalid key format:", key)
		http.Error(writer, "Invalid key", http.StatusBadRequest)
		return
	}
	origin, found := cache[key]
	if !found {
		http.Error(writer, "Target url not found", http.StatusNotFound)
		return
	}
	fmt.Printf("Redirecting from %v to: %v \n", key, origin)
	http.Redirect(writer, req, origin, http.StatusFound)
}

func Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	if _, err := w.Write([]byte("OK")); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func Check(w http.ResponseWriter, r *http.Request) {

}
