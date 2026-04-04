package rename

import (
	"testing"
)

func TestOutputName(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		// Extension always becomes .mkv
		{"movie.avi", "movie.mkv"},
		{"movie.mp4", "movie.mkv"},
		{"movie.mkv", "movie.mkv"},

		// h264 variants
		{"Movie.h264.mp4", "Movie.h265.mkv"},
		{"Movie.H264.mp4", "Movie.H265.mkv"},
		{"Movie.h.264.mp4", "Movie.h.265.mkv"},
		{"Movie.H.264.mp4", "Movie.H.265.mkv"},

		// x264 variants
		{"Movie.x264.mkv", "Movie.x265.mkv"},
		{"Movie.X264.mkv", "Movie.X265.mkv"},
		{"Movie.x.264.mkv", "Movie.x.265.mkv"},

		// AVC variants
		{"Movie.AVC.mp4", "Movie.HEVC.mkv"},
		{"Movie.avc.mp4", "Movie.hevc.mkv"},
		{"Movie.Avc.mp4", "Movie.Hevc.mkv"},

		// Multiple tokens replaced
		{"Movie.h264.AVC.mp4", "Movie.h265.HEVC.mkv"},
		{"Movie.x264.AVC.mkv", "Movie.x265.HEVC.mkv"},

		// No codec tokens — only extension changes
		{"Some.Movie.2021.mkv", "Some.Movie.2021.mkv"},
		{"No.Codec.mp4", "No.Codec.mkv"},

		// Already HEVC — no change to stem, extension updated
		{"Movie.h265.mkv", "Movie.h265.mkv"},
		{"Movie.HEVC.mkv", "Movie.HEVC.mkv"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := OutputName(tc.input)
			if got != tc.want {
				t.Errorf("OutputName(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestOutputPath_NoCollision(t *testing.T) {
	neverExists := func(string) bool { return false }
	got := OutputPath("/media/Movies", "test.mkv", neverExists)
	want := "/media/Movies/test.mkv"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestOutputPath_Collision(t *testing.T) {
	existing := map[string]bool{
		"/media/Movies/test.mkv":   true,
		"/media/Movies/test_1.mkv": true,
	}
	checkExist := func(p string) bool { return existing[p] }
	got := OutputPath("/media/Movies", "test.mkv", checkExist)
	want := "/media/Movies/test_2.mkv"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
