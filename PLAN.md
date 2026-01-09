# Plan: nimsforestencoder

## Overview
Go package that encodes a stream of `image.Image` frames into HLS video streams. Uses ffmpeg for encoding. Designed to work with nimsforestsprites output and nimsforestsmarttv input.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ Frame Source (e.g., nimsforestsprites)                     │
│   - Produces image.Image frames                            │
└─────────────────────┬───────────────────────────────────────┘
                      │ chan image.Image
                      ▼
┌─────────────────────────────────────────────────────────────┐
│ nimsforestencoder                                          │
│   ┌─────────────┐  ┌─────────────┐  ┌────────────────┐     │
│   │ Frame Input │→ │ ffmpeg pipe │→ │ HLS Segmenter  │     │
│   │ (RGBA)      │  │ (H.264)     │  │ (.m3u8 + .ts)  │     │
│   └─────────────┘  └─────────────┘  └────────────────┘     │
│                                            ↓                │
│                                     ┌────────────────┐     │
│                                     │ HTTP Server    │     │
│                                     │ (serves HLS)   │     │
│                                     └────────────────┘     │
└─────────────────────────────────────────────────────────────┘
                      │ HLS URL
                      ▼
┌─────────────────────────────────────────────────────────────┐
│ Consumer (e.g., nimsforestsmarttv, browser, VLC)           │
└─────────────────────────────────────────────────────────────┘
```

## Package Structure

```
nimsforestencoder/
├── encoder.go        # Main Encoder struct and API
├── ffmpeg.go         # ffmpeg process management
├── hls.go            # HLS segment management and serving
├── options.go        # Configuration options
├── demo/
│   └── main.go       # Demo: generates test frames and encodes
├── go.mod
├── go.sum
├── LICENSE
├── PLAN.md
└── README.md
```

## Core API Design

```go
package nimsforestencoder

// Encoder encodes image frames to HLS stream
type Encoder struct {
    // ...
}

// Options configures the encoder
type Options struct {
    Width      int           // Frame width (default 1920)
    Height     int           // Frame height (default 1080)
    FrameRate  int           // Frames per second (default 30)
    SegmentDuration int      // HLS segment duration in seconds (default 2)
    Port       int           // HTTP server port (0 = auto)
}

// Key methods:
func New(opts Options) (*Encoder, error)
func (e *Encoder) Start(ctx context.Context, frames <-chan image.Image) (hlsURL string, err error)
func (e *Encoder) Stop() error
func (e *Encoder) URL() string  // Returns HLS stream URL
```

## Implementation Steps

### 1. Initialize repository
- **First commit**: This PLAN.md
- Initialize Go module: `github.com/nimsforest/nimsforestencoder`
- Add LICENSE (MIT) and README

### 2. ffmpeg process management (`ffmpeg.go`)
- Spawn ffmpeg with stdin pipe for raw RGBA frames
- Configure H.264 encoding for HLS compatibility
- Output HLS segments to temp directory
- Handle process lifecycle

ffmpeg command:
```bash
ffmpeg -f rawvideo -pix_fmt rgba -s 1920x1080 -r 30 -i pipe:0 \
  -c:v libx264 -preset ultrafast -tune zerolatency \
  -f hls -hls_time 2 -hls_list_size 5 -hls_flags delete_segments \
  output.m3u8
```

### 3. HLS serving (`hls.go`)
- HTTP server for .m3u8 playlist and .ts segments
- Serve from temp directory
- Auto-cleanup old segments

### 4. Encoder API (`encoder.go`)
- Accept `chan image.Image`
- Convert each frame to raw RGBA bytes
- Write to ffmpeg stdin
- Manage goroutines for frame processing

### 5. Demo (`demo/main.go`)
- Generate animated test frames (moving colored rectangle)
- Encode to HLS
- Print URL for testing with VLC or TV

## Dependencies
- ffmpeg (external, must be installed)
- Standard library only for Go code

## Validation (without nimsforest2)
1. Run demo: `go run ./demo`
2. Open HLS URL in VLC: `vlc http://localhost:PORT/stream.m3u8`
3. Or send to TV via nimsforestsmarttv

## Integration with nimsforestsprites
```go
sprites := nimsforestsprites.NewRenderer(assets)
frames := sprites.Frames(ctx, viewModel)  // chan image.Image

encoder, _ := nimsforestencoder.New(nimsforestencoder.Options{
    Width: 1920, Height: 1080, FrameRate: 30,
})
hlsURL, _ := encoder.Start(ctx, frames)

// hlsURL can be sent to nimsforestsmarttv
```
