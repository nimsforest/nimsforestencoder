package nimsforestencoder

import (
	"context"
	"fmt"
	"image"
	"image/draw"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Encoder encodes image frames to HLS stream.
type Encoder struct {
	opts      Options
	ffmpeg    *ffmpegProcess
	hlsServer *hlsServer
	outputDir string

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// New creates a new Encoder with the given options.
func New(opts Options) (*Encoder, error) {
	opts = opts.withDefaults()

	return &Encoder{
		opts: opts,
	}, nil
}

// Start begins encoding frames from the channel and returns the HLS URL.
// It starts the ffmpeg process and HTTP server.
func (e *Encoder) Start(ctx context.Context, frames <-chan image.Image) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return "", fmt.Errorf("encoder already running")
	}

	// Create temp directory for HLS output
	outputDir, err := os.MkdirTemp("", "nimsforestencoder-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	e.outputDir = outputDir

	// Start HLS server first so we know the port
	hlsServer, err := newHLSServer(outputDir, e.opts.Port)
	if err != nil {
		os.RemoveAll(outputDir)
		return "", fmt.Errorf("failed to create HLS server: %w", err)
	}
	e.hlsServer = hlsServer
	hlsServer.Start()

	// Start ffmpeg process
	ffmpeg, err := newFFmpegProcess(outputDir, e.opts)
	if err != nil {
		hlsServer.Stop(context.Background())
		os.RemoveAll(outputDir)
		return "", fmt.Errorf("failed to start ffmpeg: %w", err)
	}
	e.ffmpeg = ffmpeg

	// Create cancellable context for frame processing
	ctx, cancel := context.WithCancel(ctx)
	e.cancel = cancel
	e.running = true

	// Start frame processing goroutine
	e.wg.Add(1)
	go e.processFrames(ctx, frames)

	return hlsServer.URL(), nil
}

// processFrames reads frames from the channel and writes them to ffmpeg.
func (e *Encoder) processFrames(ctx context.Context, frames <-chan image.Image) {
	defer e.wg.Done()

	// Buffer for RGBA data
	bufSize := e.opts.Width * e.opts.Height * 4
	buf := make([]byte, bufSize)

	for {
		select {
		case <-ctx.Done():
			return
		case frame, ok := <-frames:
			if !ok {
				// Channel closed, stop processing
				return
			}

			// Convert frame to RGBA bytes
			if err := e.frameToRGBA(frame, buf); err != nil {
				// Log error but continue processing
				continue
			}

			// Write to ffmpeg
			if err := e.ffmpeg.WriteFrame(buf); err != nil {
				// ffmpeg may have exited
				return
			}
		}
	}
}

// frameToRGBA converts an image.Image to raw RGBA bytes.
func (e *Encoder) frameToRGBA(img image.Image, buf []byte) error {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Validate dimensions
	if width != e.opts.Width || height != e.opts.Height {
		return fmt.Errorf("frame size mismatch: got %dx%d, expected %dx%d",
			width, height, e.opts.Width, e.opts.Height)
	}

	// Try to use existing RGBA data if available
	if rgba, ok := img.(*image.RGBA); ok && rgba.Stride == width*4 {
		copy(buf, rgba.Pix)
		return nil
	}

	// Convert to RGBA
	rgba := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(rgba, rgba.Bounds(), img, bounds.Min, draw.Src)
	copy(buf, rgba.Pix)

	return nil
}

// Stop stops the encoder, closes ffmpeg, and shuts down the HTTP server.
func (e *Encoder) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return nil
	}

	// Signal frame processing to stop
	if e.cancel != nil {
		e.cancel()
	}

	// Wait for frame processing to finish
	e.wg.Wait()

	var errs []error

	// Close ffmpeg (this will finalize the stream)
	if e.ffmpeg != nil {
		if err := e.ffmpeg.Close(); err != nil {
			errs = append(errs, fmt.Errorf("ffmpeg close: %w", err))
		}
	}

	// Stop HTTP server
	if e.hlsServer != nil {
		if err := e.hlsServer.Stop(context.Background()); err != nil {
			errs = append(errs, fmt.Errorf("HLS server stop: %w", err))
		}
	}

	// Clean up temp directory
	if e.outputDir != "" {
		if err := os.RemoveAll(e.outputDir); err != nil {
			errs = append(errs, fmt.Errorf("cleanup: %w", err))
		}
	}

	e.running = false

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// URL returns the HLS stream URL. Only valid after Start() is called.
func (e *Encoder) URL() string {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.hlsServer != nil {
		return e.hlsServer.URL()
	}
	return ""
}

// WaitReady waits for the HLS stream to be ready (first segment created).
// Returns an error if the timeout is exceeded or context is cancelled.
func (e *Encoder) WaitReady(ctx context.Context, timeout time.Duration) error {
	e.mu.Lock()
	outputDir := e.outputDir
	e.mu.Unlock()

	if outputDir == "" {
		return fmt.Errorf("encoder not started")
	}

	m3u8Path := filepath.Join(outputDir, "stream.m3u8")
	deadline := time.Now().Add(timeout)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for HLS stream to be ready")
		}

		// Check if m3u8 file exists and has content
		info, err := os.Stat(m3u8Path)
		if err == nil && info.Size() > 0 {
			return nil
		}

		time.Sleep(100 * time.Millisecond)
	}
}
