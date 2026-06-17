package enricher

// Segment is one timestamped caption line (seconds).
type Segment struct {
	Index int     `json:"index"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

// Frame is one extracted keyframe with the transcript spoken around its timestamp.
type Frame struct {
	File        string  `json:"file"` // path relative to the bundle dir, e.g. frames/0001.jpg
	T           float64 `json:"t"`    // seconds into the video
	NearestText string  `json:"nearest_text"`
}

// Bundle is the on-disk artifact a multimodal model reads to understand a video.
type Bundle struct {
	VideoID  string    `json:"video_id"`
	Title    string    `json:"title"`
	URL      string    `json:"url"`
	Duration float64   `json:"duration"`
	Dir      string    `json:"-"` // absolute bundle directory
	Segments []Segment `json:"segments"`
	Frames   []Frame   `json:"frames"`
}
