package main

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestShortenWithGetBody(t *testing.T) {
	url := "https://url.com"
	req := httptest.NewRequest("GET", "/shorten", strings.NewReader(url))
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()
	Shorten(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("got status %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.HasPrefix(body, BaseURL) {
		t.Error("response does not start with base URL")
	}
	if ok, _ := regexp.MatchString(`.*/[a-zA-Z]{6}$`, body); !ok {
		t.Error("invalid key format")
	}
}

func TestShortenValidPost(t *testing.T) {

	url := "https://example.com"
	req := httptest.NewRequest("POST", "/shorten", strings.NewReader(url))
	rr := httptest.NewRecorder()
	Shorten(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("got status %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.HasPrefix(body, BaseURL) {
		t.Error("response does not start with base URL")
	}
	if ok, _ := regexp.MatchString(`.*/[a-zA-Z]{6}$`, body); !ok {
		t.Error("invalid key format")
	}
}

func TestShortenInvalidUrl(t *testing.T) {

	req := httptest.NewRequest("POST", "/shorten", strings.NewReader("not-a-url"))
	rr := httptest.NewRecorder()
	Shorten(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Invalid URL") {
		t.Error("missing invalid URL message")
	}
}

func TestInvalidPathRandomRandom(t *testing.T) {

	req := httptest.NewRequest("GET", "/random/random", nil)
	rr := httptest.NewRecorder()
	Redirect(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestInvalidPathRandom(t *testing.T) {

	req := httptest.NewRequest("GET", "/a/b/c", nil)
	rr := httptest.NewRecorder()
	Redirect(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestWrongShortenPath(t *testing.T) {

	req := httptest.NewRequest("GET", "/shorten/1234567890", nil)
	rr := httptest.NewRecorder()
	Shorten(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestWrongShortenPathWithQuery(t *testing.T) {

	req := httptest.NewRequest("GET", "/shorten/1234567890?url=https://example.com", nil)
	rr := httptest.NewRecorder()
	Shorten(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestRedirectFound(t *testing.T) {
	// create shortened
	body := strings.NewReader("https://openai.com")
	createReq := httptest.NewRequest("POST", "/shorten", body)
	r1 := httptest.NewRecorder()
	Shorten(r1, createReq)
	key := strings.TrimPrefix(r1.Body.String(), BaseURL)

	req := httptest.NewRequest("GET", "/{key}", nil)
	req.SetPathValue("key", key)
	rr := httptest.NewRecorder()
	Redirect(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rr.Code)
	}
	loc := rr.Header().Get("Location")
	if loc != "https://openai.com" {
		t.Errorf("expected redirect to https://openai.com, got %s", loc)
	}
}

func TestRedirectNotFound(t *testing.T) {

	req := httptest.NewRequest("GET", "/{key}", nil)
	req.SetPathValue("key", "foobar")
	rr := httptest.NewRecorder()
	Redirect(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Target url not found") {
		t.Error("missing not found message")
	}
}

func TestHealthEndpoint(t *testing.T) {
	// Health may return OK or ServiceUnavailable
	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()
	Health(rr, req)
	if rr.Code != http.StatusOK && rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("unexpected status %d", rr.Code)
	}
	body := rr.Body.String()
	if rr.Code == http.StatusOK {
		if body != "OK" {
			t.Error("expected OK body")
		}
	} else {
		if !strings.Contains(body, "Health check failed") {
			t.Error("missing health failure message")
		}
	}
}

func TestCheckValidShortenedUrl(t *testing.T) {
	// create
	body := strings.NewReader("https://github.com")
	createReq := httptest.NewRequest("POST", "/shorten", body)
	r1 := httptest.NewRecorder()
	Shorten(r1, createReq)
	short := r1.Body.String()
	// check
	req := httptest.NewRequest("POST", "/check", strings.NewReader(short))
	rr := httptest.NewRecorder()
	Check(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if rr.Body.String() != "https://github.com" {
		t.Error("unexpected body")
	}
}

func TestCheckInvalidUrl(t *testing.T) {
	body := strings.NewReader("not-a-valid-url")
	req := httptest.NewRequest("POST", "/check", body)
	rr := httptest.NewRecorder()
	Check(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Invalid URL") {
		t.Error("missing invalid URL message")
	}
}

func TestCheckThirdPartyUrl(t *testing.T) {
	body := strings.NewReader("https://example.com/abc123")
	req := httptest.NewRequest("POST", "/check", body)
	rr := httptest.NewRecorder()
	Check(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "not yet supported") {
		t.Error("missing third-party support message")
	}
}

func TestCheckNonExistentKey(t *testing.T) {
	req := httptest.NewRequest("POST", "/check", strings.NewReader(BaseURL+"notfnd"))
	rr := httptest.NewRecorder()
	Check(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Target url not found") {
		t.Error("missing target not found message")
	}
}

func TestCheckEmptyBody(t *testing.T) {
	req := httptest.NewRequest("POST", "/check", strings.NewReader(""))
	rr := httptest.NewRecorder()
	Check(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCheckUrlWithoutScheme(t *testing.T) {
	req := httptest.NewRequest("POST", "/check", strings.NewReader("url.radhi.tech/abc123"))
	rr := httptest.NewRecorder()
	Check(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCheckUrlWithoutHost(t *testing.T) {
	req := httptest.NewRequest("POST", "/check", strings.NewReader("https:///abc123"))
	rr := httptest.NewRecorder()
	Check(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCheckWithGetMethod(t *testing.T) {
	req := httptest.NewRequest("GET", "/check", strings.NewReader(BaseURL+"/abc123"))
	rr := httptest.NewRecorder()
	Check(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}
