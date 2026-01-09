package nimsforestencoder

// Options configures the encoder.
type Options struct {
	// Width is the frame width in pixels. Default: 1920
	Width int

	// Height is the frame height in pixels. Default: 1080
	Height int

	// FrameRate is the frames per second. Default: 30
	FrameRate int

	// SegmentDuration is the HLS segment duration in seconds. Default: 2
	SegmentDuration int

	// Port is the HTTP server port. 0 means auto-assign. Default: 0
	Port int
}

// DefaultOptions returns Options with default values.
func DefaultOptions() Options {
	return Options{
		Width:           1920,
		Height:          1080,
		FrameRate:       30,
		SegmentDuration: 2,
		Port:            0,
	}
}

// withDefaults returns a copy of opts with default values applied for zero fields.
func (opts Options) withDefaults() Options {
	defaults := DefaultOptions()

	if opts.Width == 0 {
		opts.Width = defaults.Width
	}
	if opts.Height == 0 {
		opts.Height = defaults.Height
	}
	if opts.FrameRate == 0 {
		opts.FrameRate = defaults.FrameRate
	}
	if opts.SegmentDuration == 0 {
		opts.SegmentDuration = defaults.SegmentDuration
	}
	// Port 0 is valid (auto-assign), so we don't apply default

	return opts
}
