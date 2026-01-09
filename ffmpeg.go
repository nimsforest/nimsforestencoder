package nimsforestencoder

import (
	"fmt"
	"io"
	"os/exec"
	"strconv"
)

// ffmpegProcess manages an ffmpeg subprocess for encoding raw RGBA frames to HLS.
type ffmpegProcess struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	outputDir string
	opts      Options
}

// newFFmpegProcess creates and starts a new ffmpeg process.
// It accepts raw RGBA frames on stdin and outputs HLS segments to outputDir.
func newFFmpegProcess(outputDir string, opts Options) (*ffmpegProcess, error) {
	// Build ffmpeg command
	// ffmpeg -f rawvideo -pix_fmt rgba -s WxH -r FPS -i pipe:0 \
	//   -c:v libx264 -preset ultrafast -tune zerolatency \
	//   -f hls -hls_time SEGMENT_DURATION -hls_list_size 5 -hls_flags delete_segments \
	//   OUTPUT_DIR/stream.m3u8

	resolution := fmt.Sprintf("%dx%d", opts.Width, opts.Height)
	frameRate := strconv.Itoa(opts.FrameRate)
	segmentDuration := strconv.Itoa(opts.SegmentDuration)
	outputPath := outputDir + "/stream.m3u8"

	args := []string{
		"-f", "rawvideo",
		"-pix_fmt", "rgba",
		"-s", resolution,
		"-r", frameRate,
		"-i", "pipe:0",
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-tune", "zerolatency",
		"-pix_fmt", "yuv420p", // Required for compatibility
		"-f", "hls",
		"-hls_time", segmentDuration,
		"-hls_list_size", "5",
		"-hls_flags", "delete_segments",
		outputPath,
	}

	cmd := exec.Command("ffmpeg", args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		stdin.Close()
		return nil, fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	return &ffmpegProcess{
		cmd:       cmd,
		stdin:     stdin,
		outputDir: outputDir,
		opts:      opts,
	}, nil
}

// WriteFrame writes raw RGBA frame data to ffmpeg.
// The data must be exactly Width * Height * 4 bytes (RGBA).
func (f *ffmpegProcess) WriteFrame(data []byte) error {
	expectedSize := f.opts.Width * f.opts.Height * 4
	if len(data) != expectedSize {
		return fmt.Errorf("invalid frame size: got %d, expected %d", len(data), expectedSize)
	}

	_, err := f.stdin.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write frame: %w", err)
	}

	return nil
}

// Close closes the stdin pipe and waits for ffmpeg to finish.
func (f *ffmpegProcess) Close() error {
	if err := f.stdin.Close(); err != nil {
		return fmt.Errorf("failed to close stdin: %w", err)
	}

	if err := f.cmd.Wait(); err != nil {
		return fmt.Errorf("ffmpeg exited with error: %w", err)
	}

	return nil
}

// Kill forcefully terminates the ffmpeg process.
func (f *ffmpegProcess) Kill() error {
	if f.cmd.Process != nil {
		return f.cmd.Process.Kill()
	}
	return nil
}
