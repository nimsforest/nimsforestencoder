# nimsforestencoder

Go package that encodes a stream of `image.Image` frames into HLS video streams. Uses ffmpeg for encoding.

## Features

- Accepts `chan image.Image` as input
- Pipes raw RGBA frames to ffmpeg
- Outputs HLS segments (.m3u8 + .ts files)
- Built-in HTTP server to serve HLS stream
- Standard library only (ffmpeg is external dependency)

## Installation

```bash
go get github.com/nimsforest/nimsforestencoder
```

## Requirements

- Go 1.21+
- ffmpeg with libx264 support

## Usage

```go
package main

import (
    "context"
    "image"
    "log"

    "github.com/nimsforest/nimsforestencoder"
)

func main() {
    ctx := context.Background()

    // Create frame channel
    frames := make(chan image.Image)

    // Create encoder
    encoder, err := nimsforestencoder.New(nimsforestencoder.Options{
        Width:     1920,
        Height:    1080,
        FrameRate: 30,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Start encoding
    hlsURL, err := encoder.Start(ctx, frames)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("HLS stream available at: %s", hlsURL)

    // Send frames...
    // frames <- yourImage

    // Stop encoder when done
    encoder.Stop()
}
```

## Demo

Run the included demo that generates animated test frames:

```bash
go run ./demo
```

Then open the printed HLS URL in VLC:

```bash
vlc http://localhost:PORT/stream.m3u8
```

## Configuration

| Option | Default | Description |
|--------|---------|-------------|
| Width | 1920 | Frame width in pixels |
| Height | 1080 | Frame height in pixels |
| FrameRate | 30 | Frames per second |
| SegmentDuration | 2 | HLS segment duration in seconds |
| Port | 0 | HTTP server port (0 = auto-assign) |

## Architecture

```
Frame Source -> nimsforestencoder -> HLS Stream -> Consumer
                     |
                     v
               +-----------+
               | ffmpeg    |
               | (H.264)   |
               +-----------+
                     |
                     v
               +-----------+
               | HLS Files |
               | (.m3u8)   |
               +-----------+
                     |
                     v
               +-----------+
               | HTTP      |
               | Server    |
               +-----------+
```

## License

MIT License - see [LICENSE](LICENSE) for details.
