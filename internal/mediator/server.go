package mediator

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/tabctl/tabctl/internal/config"
	"github.com/tabctl/tabctl/internal/utils"
	"github.com/tabctl/tabctl/pkg/types"
)

// Server represents the HTTP server for the mediator
type Server struct {
	config    *config.MediatorConfig
	router    *mux.Router
	server    *http.Server
	remoteAPI *RemoteAPI
}

// NewServer creates a new HTTP server
func NewServer(cfg *config.MediatorConfig, remoteAPI *RemoteAPI) *Server {
	s := &Server{
		config:    cfg,
		remoteAPI: remoteAPI,
	}

	s.setupRoutes()

	s.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler: s.router,
	}

	return s
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	s.router = mux.NewRouter()

	// Add CORS middleware
	s.router.Use(corsMiddleware)

	// Register routes
	s.router.HandleFunc("/", s.rootHandler).Methods("GET")
	s.router.HandleFunc("/shutdown", s.shutdownHandler).Methods("GET")
	s.router.HandleFunc("/list_tabs", s.listTabsHandler).Methods("GET")
	s.router.HandleFunc("/query_tabs/{query_info}", s.queryTabsHandler).Methods("GET")
	s.router.HandleFunc("/move_tabs/{move_triplets}", s.moveTabsHandler).Methods("GET")
	s.router.HandleFunc("/open_urls/{window_id}", s.openURLsHandler).Methods("POST")
	s.router.HandleFunc("/open_urls", s.openURLsHandler).Methods("POST")
	s.router.HandleFunc("/update_tabs", s.updateTabsHandler).Methods("POST")
	s.router.HandleFunc("/close_tabs/{tab_ids}", s.closeTabsHandler).Methods("GET")
	s.router.HandleFunc("/new_tab/{query}", s.newTabHandler).Methods("GET")
	s.router.HandleFunc("/activate_tab/{tab_id}", s.activateTabHandler).Methods("GET")
	s.router.HandleFunc("/get_active_tabs", s.getActiveTabsHandler).Methods("GET")
	s.router.HandleFunc("/get_screenshot", s.getScreenshotHandler).Methods("GET")
	s.router.HandleFunc("/get_words/{tab_id}", s.getWordsHandler).Methods("GET")
	s.router.HandleFunc("/get_words", s.getWordsHandler).Methods("GET")
	s.router.HandleFunc("/get_text", s.getTextHandler).Methods("GET")
	s.router.HandleFunc("/get_html", s.getHTMLHandler).Methods("GET")
	s.router.HandleFunc("/get_pid", s.getPIDHandler).Methods("GET")
	s.router.HandleFunc("/get_browser", s.getBrowserHandler).Methods("GET")
	s.router.HandleFunc("/echo", s.echoHandler).Methods("GET")
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("Starting HTTP server on %s", s.server.Addr)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	log.Printf("Shutting down HTTP server")
	// TODO: Implement graceful shutdown with context
	return nil
}

// Handler implementations

func (s *Server) rootHandler(w http.ResponseWriter, r *http.Request) {
	var links []string
	s.router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		path, _ := route.GetPathTemplate()
		methods, _ := route.GetMethods()
		if path != "" {
			links = append(links, fmt.Sprintf("%s\t%s", strings.Join(methods, ","), path))
		}
		return nil
	})
	w.Write([]byte(strings.Join(links, "\n")))
}

func (s *Server) shutdownHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
	go func() {
		s.Shutdown()
	}()
}

func (s *Server) listTabsHandler(w http.ResponseWriter, r *http.Request) {
	tabLines, err := s.remoteAPI.ListTabs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert TSV lines to Tab objects
	var tabs []types.Tab
	for _, line := range tabLines {
		tab, err := utils.ParseTabLine(line)
		if err != nil {
			continue // Skip invalid lines
		}
		tabs = append(tabs, tab)
	}

	// Return JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tabs)
}

func (s *Server) queryTabsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	queryInfo := vars["query_info"]

	tabs, err := s.remoteAPI.QueryTabs(queryInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(strings.Join(tabs, "\n")))
}

func (s *Server) moveTabsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	moveTriplets := vars["move_triplets"]

	// URL decode the triplets
	decoded, err := url.QueryUnescape(moveTriplets)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := s.remoteAPI.MoveTabs(decoded)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(result))
}

func (s *Server) openURLsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	windowIDStr := vars["window_id"]

	var windowID *int
	if windowIDStr != "" {
		id, err := strconv.Atoi(windowIDStr)
		if err != nil {
			http.Error(w, "Invalid window_id", http.StatusBadRequest)
			return
		}
		windowID = &id
	}

	// Read URLs from multipart form or request body
	var urls []string

	if err := r.ParseMultipartForm(10 << 20); err == nil {
		// Try multipart form
		file, _, err := r.FormFile("urls")
		if err == nil {
			defer file.Close()
			content, _ := io.ReadAll(file)
			urls = strings.Split(string(content), "\n")
		}
	} else {
		// Try raw body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Cannot read URLs", http.StatusBadRequest)
			return
		}
		urls = strings.Split(string(body), "\n")
	}

	// Clean up URLs
	var cleanURLs []string
	for _, url := range urls {
		url = strings.TrimSpace(url)
		if url != "" {
			cleanURLs = append(cleanURLs, url)
		}
	}

	result, err := s.remoteAPI.OpenURLs(cleanURLs, windowID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(strings.Join(result, "\n")))
}

func (s *Server) updateTabsHandler(w http.ResponseWriter, r *http.Request) {
	// Read updates from multipart form or request body
	var updates []map[string]interface{}

	if err := r.ParseMultipartForm(10 << 20); err == nil {
		// Try multipart form
		file, _, err := r.FormFile("updates")
		if err == nil {
			defer file.Close()
			if err := json.NewDecoder(file).Decode(&updates); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}
		}
	} else {
		// Try raw body
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
	}

	result, err := s.remoteAPI.UpdateTabs(updates)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(strings.Join(result, "\n")))
}

func (s *Server) closeTabsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tabIDs := vars["tab_ids"]

	result, err := s.remoteAPI.CloseTabs(tabIDs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(result))
}

func (s *Server) newTabHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	query := vars["query"]

	result, err := s.remoteAPI.NewTab(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(result))
}

func (s *Server) activateTabHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tabIDStr := vars["tab_id"]

	tabID, err := strconv.Atoi(tabIDStr)
	if err != nil {
		http.Error(w, "Invalid tab_id", http.StatusBadRequest)
		return
	}

	focused := r.URL.Query().Get("focused") == "true"

	if err := s.remoteAPI.ActivateTab(tabID, focused); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte("OK"))
}

func (s *Server) getActiveTabsHandler(w http.ResponseWriter, r *http.Request) {
	result, err := s.remoteAPI.GetActiveTabs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(result))
}

func (s *Server) getScreenshotHandler(w http.ResponseWriter, r *http.Request) {
	result, err := s.remoteAPI.GetScreenshot()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(result))
}

func (s *Server) getWordsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tabIDStr := vars["tab_id"]

	var tabID *int
	if tabIDStr != "" {
		id, err := strconv.Atoi(tabIDStr)
		if err == nil {
			tabID = &id
		}
	}

	matchRegex := r.URL.Query().Get("match_regex")
	if matchRegex == "" {
		matchRegex = config.DefaultGetWordsMatchRegex
	}

	joinWith := r.URL.Query().Get("join_with")
	if joinWith == "" {
		joinWith = config.DefaultGetWordsJoinWith
	}

	// URL decode parameters
	matchRegex, _ = url.QueryUnescape(matchRegex)
	joinWith, _ = url.QueryUnescape(joinWith)

	words, err := s.remoteAPI.GetWords(tabID, matchRegex, joinWith)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(strings.Join(words, "\n")))
}

func (s *Server) getTextHandler(w http.ResponseWriter, r *http.Request) {
	delimiterRegex := r.URL.Query().Get("delimiter_regex")
	if delimiterRegex == "" {
		delimiterRegex = config.DefaultGetTextDelimiterRegex
	}

	replaceWith := r.URL.Query().Get("replace_with")
	if replaceWith == "" {
		replaceWith = config.DefaultGetTextReplaceWith
	}

	// URL decode parameters
	delimiterRegex, _ = url.QueryUnescape(delimiterRegex)
	replaceWith, _ = url.QueryUnescape(replaceWith)

	lines, err := s.remoteAPI.GetText(delimiterRegex, replaceWith)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(strings.Join(lines, "\n")))
}

func (s *Server) getHTMLHandler(w http.ResponseWriter, r *http.Request) {
	delimiterRegex := r.URL.Query().Get("delimiter_regex")
	if delimiterRegex == "" {
		delimiterRegex = config.DefaultGetHTMLDelimiterRegex
	}

	replaceWith := r.URL.Query().Get("replace_with")
	if replaceWith == "" {
		replaceWith = config.DefaultGetHTMLReplaceWith
	}

	// URL decode parameters
	delimiterRegex, _ = url.QueryUnescape(delimiterRegex)
	replaceWith, _ = url.QueryUnescape(replaceWith)

	lines, err := s.remoteAPI.GetHTML(delimiterRegex, replaceWith)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(strings.Join(lines, "\n")))
}

func (s *Server) getPIDHandler(w http.ResponseWriter, r *http.Request) {
	pid := s.remoteAPI.GetPID()
	w.Write([]byte(strconv.Itoa(pid)))
}

func (s *Server) getBrowserHandler(w http.ResponseWriter, r *http.Request) {
	browser := s.remoteAPI.GetBrowser()
	w.Write([]byte(browser))
}

func (s *Server) echoHandler(w http.ResponseWriter, r *http.Request) {
	title := r.URL.Query().Get("title")
	if title == "" {
		title = "title"
	}

	body := r.URL.Query().Get("body")
	if body == "" {
		body = "body"
	}

	html := fmt.Sprintf("<html><head><title>%s</title></head><body>%s</body></html>", title, body)
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// corsMiddleware adds CORS headers to responses
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}