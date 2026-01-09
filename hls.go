package nimsforestencoder

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
)

// hlsServer serves HLS segments over HTTP.
type hlsServer struct {
	server     *http.Server
	listener   net.Listener
	outputDir  string
	actualPort int
}

// newHLSServer creates a new HLS HTTP server.
func newHLSServer(outputDir string, port int) (*hlsServer, error) {
	// Create listener first to get actual port if port is 0
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	// Get the actual port assigned
	actualPort := listener.Addr().(*net.TCPAddr).Port

	// Create file server for the output directory
	mux := http.NewServeMux()

	// Serve HLS files with proper MIME types
	fileServer := http.FileServer(http.Dir(outputDir))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Set appropriate headers for HLS
		ext := filepath.Ext(r.URL.Path)
		switch ext {
		case ".m3u8":
			w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		case ".ts":
			w.Header().Set("Content-Type", "video/mp2t")
		}

		// Allow CORS for browser playback
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Disable caching for live stream
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

		fileServer.ServeHTTP(w, r)
	})

	server := &http.Server{
		Handler: mux,
	}

	return &hlsServer{
		server:     server,
		listener:   listener,
		outputDir:  outputDir,
		actualPort: actualPort,
	}, nil
}

// Start starts the HTTP server in a goroutine.
func (h *hlsServer) Start() {
	go func() {
		// Serve will return when the listener is closed
		_ = h.server.Serve(h.listener)
	}()
}

// Stop gracefully shuts down the HTTP server.
func (h *hlsServer) Stop(ctx context.Context) error {
	return h.server.Shutdown(ctx)
}

// Port returns the actual port the server is listening on.
func (h *hlsServer) Port() int {
	return h.actualPort
}

// URL returns the full URL to the HLS playlist.
func (h *hlsServer) URL() string {
	ip := getOutboundIP()
	return fmt.Sprintf("http://%s:%d/stream.m3u8", ip, h.actualPort)
}

// getOutboundIP gets the preferred outbound IP address
func getOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "localhost"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}
