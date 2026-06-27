package daemon

import (
	"context"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/kamichidu/go-netclip/internal/clipboard"
)

// Server coordinates the HTTP daemon listening on a port to receive copies.
type Server struct {
	addr             string
	clipboardCommand string
	httpServer       *http.Server
}

// NewServer creates a new Server instance.
type Listener interface {
	Addr() net.Addr
}

func NewServer(addr string, clipboardCommand string) *Server {
	return &Server{
		addr:             addr,
		clipboardCommand: clipboardCommand,
	}
}

// Start boots up the HTTP server. It is blocking.
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/copy", s.handleCopy)

	s.httpServer = &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	log.Printf("Starting netclip daemon on %s", s.addr)
	return s.httpServer.ListenAndServe()
}

// StartWithListener starts the HTTP server using an existing net.Listener. Helpful for testing.
func (s *Server) StartWithListener(ln net.Listener) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/copy", s.handleCopy)

	s.httpServer = &http.Server{
		Handler: mux,
	}

	log.Printf("Starting netclip daemon on listener %s", ln.Addr())
	return s.httpServer.Serve(ln)
}

// Shutdown stops the server gracefully.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

func (s *Server) handleCopy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	adapter, err := clipboard.GetAdapter(s.clipboardCommand)
	if err != nil {
		log.Printf("Clipboard adapter error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := adapter.Write(ctx, body); err != nil {
		log.Printf("Clipboard write error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
