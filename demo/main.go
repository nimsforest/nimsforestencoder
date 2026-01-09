// Demo program that generates animated test frames and encodes them to HLS.
// Run with: go run ./demo
// Then open the printed URL in VLC: vlc http://localhost:PORT/stream.m3u8
package main

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nimsforest/nimsforestencoder"
)

const (
	width     = 1280
	height    = 720
	frameRate = 30

	// Rectangle properties
	rectWidth  = 100
	rectHeight = 100
	speed      = 5 // pixels per frame
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nShutting down...")
		cancel()
	}()

	// Create frame channel
	frames := make(chan image.Image, 2)

	// Create encoder
	encoder, err := nimsforestencoder.New(nimsforestencoder.Options{
		Width:     width,
		Height:    height,
		FrameRate: frameRate,
	})
	if err != nil {
		log.Fatalf("Failed to create encoder: %v", err)
	}

	// Start encoding
	hlsURL, err := encoder.Start(ctx, frames)
	if err != nil {
		log.Fatalf("Failed to start encoder: %v", err)
	}

	fmt.Println("========================================")
	fmt.Println("HLS stream is now available!")
	fmt.Printf("URL: %s\n", hlsURL)
	fmt.Println("")
	fmt.Println("Open in VLC:")
	fmt.Printf("  vlc %s\n", hlsURL)
	fmt.Println("")
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println("========================================")

	// Generate and send frames
	generateFrames(ctx, frames)

	// Stop encoder
	if err := encoder.Stop(); err != nil {
		log.Printf("Error stopping encoder: %v", err)
	}

	fmt.Println("Demo finished")
}

// generateFrames creates animated frames with a moving rectangle.
func generateFrames(ctx context.Context, frames chan<- image.Image) {
	defer close(frames)

	ticker := time.NewTicker(time.Second / frameRate)
	defer ticker.Stop()

	// Rectangle position
	x := 0
	y := height / 2 - rectHeight/2

	// Direction (1 = right, -1 = left)
	direction := 1

	// Colors for the animation
	bgColor := color.RGBA{R: 30, G: 30, B: 50, A: 255}     // Dark blue background
	rectColor := color.RGBA{R: 255, G: 100, B: 50, A: 255} // Orange rectangle

	frameCount := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Create frame
			frame := image.NewRGBA(image.Rect(0, 0, width, height))

			// Fill background
			for py := 0; py < height; py++ {
				for px := 0; px < width; px++ {
					frame.SetRGBA(px, py, bgColor)
				}
			}

			// Draw rectangle
			for py := y; py < y+rectHeight && py < height; py++ {
				for px := x; px < x+rectWidth && px < width; px++ {
					if px >= 0 && py >= 0 {
						frame.SetRGBA(px, py, rectColor)
					}
				}
			}

			// Add a simple pattern that changes with frame count for visual feedback
			// Draw frame counter indicator (vertical bar that moves)
			indicatorX := (frameCount * 2) % width
			for py := 0; py < 10; py++ {
				frame.SetRGBA(indicatorX, py, color.RGBA{R: 255, G: 255, B: 255, A: 255})
			}

			// Send frame
			select {
			case frames <- frame:
			case <-ctx.Done():
				return
			}

			// Update rectangle position
			x += speed * direction
			if x+rectWidth >= width {
				direction = -1
			} else if x <= 0 {
				direction = 1
			}

			frameCount++

			// Print progress every second
			if frameCount%frameRate == 0 {
				fmt.Printf("Generated %d frames (%d seconds)\n", frameCount, frameCount/frameRate)
			}
		}
	}
}
