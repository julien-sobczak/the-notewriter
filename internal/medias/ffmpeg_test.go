package medias

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain allows the test binary to impersonate many other binaries
// using the same technique as used by Golang in src/os/exec_test.go.
// Read the official tests or this article https://abhinavg.net/2022/05/15/hijack-testmain/.
func TestMain(m *testing.M) {
	// See https://abhinavg.net/2022/05/15/hijack-testmain/ for inspiration
	behavior := os.Getenv("TEST_BEHAVIOR")
	switch behavior {
	case "":
		os.Exit(m.Run())
	// errors
	case "dump_cmd":
		dump_cmd()
	default:
		log.Fatalf("unknown behavior %q", behavior)
	}
}

func dump_cmd() {
	// We write the target file but instead of converting the media,
	// we simply output the command so that tests can check the correct
	// arguments are correctly passed.

	// We consider the last argument to be the target file.
	dest := os.Args[len(os.Args)-1]
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		log.Fatalf("file already exists %s: %v", dest, err)
	}
	cmd := "ffmpeg " + strings.Join(os.Args[1:], " ")
	err := os.WriteFile(dest, []byte(cmd), 0644)
	if err != nil {
		log.Fatalf("file already exists %s: %v", dest, err)
	}
}

func TestConvertToAVIF(t *testing.T) {

	t.Run("Original", func(t *testing.T) {
		t.Setenv("TEST_BEHAVIOR", "dump_cmd")
		converter := &FFmpegConverter{
			exe: testExe(t),
		}

		mediasDir := filepath.Join("testdata", "TestMedias/medias")
		outputDir := t.TempDir()

		src := filepath.Join(mediasDir, "tree-landscape-large.jpg")
		dest := filepath.Join(outputDir, "out.avif")

		err := converter.ToAVIF(src, dest, OriginalSize())
		require.NoError(t, err)

		// Check cmd
		actual, err := os.ReadFile(dest)
		require.NoError(t, err)
		expected := fmt.Sprintf("ffmpeg -i %s %s", src, dest)
		assert.Equal(t, expected, string(actual))

	})

	t.Run("Preview", func(t *testing.T) {
		t.Setenv("TEST_BEHAVIOR", "dump_cmd")
		converter := &FFmpegConverter{
			exe: testExe(t),
		}

		mediasDir := filepath.Join("testdata", "TestMedias/medias")
		outputDir := t.TempDir()

		src := filepath.Join(mediasDir, "tree-landscape-large.jpg")
		dest := filepath.Join(outputDir, "out.avif")

		err := converter.ToAVIF(src, dest, ResizeTo(150))
		require.NoError(t, err)

		// Check cmd
		actual, err := os.ReadFile(dest)
		require.NoError(t, err)
		expected := fmt.Sprintf("ffmpeg -i %s -vf scale=150:-1 %s", src, dest)
		assert.Equal(t, expected, string(actual))
	})

	t.Run("Preview from video", func(t *testing.T) {
		t.Setenv("TEST_BEHAVIOR", "dump_cmd")
		converter := &FFmpegConverter{
			exe: testExe(t),
		}

		mediasDir := filepath.Join("testdata", "TestMedias/medias")
		outputDir := t.TempDir()

		src := filepath.Join(mediasDir, "forest-large.webm")
		dest := filepath.Join(outputDir, "out.avif")

		err := converter.ToAVIF(src, dest, ResizeTo(150))
		require.NoError(t, err)

		// Check cmd
		actual, err := os.ReadFile(dest)
		require.NoError(t, err)
		expected := fmt.Sprintf("ffmpeg -i %s -vf select=eq(n\\,0),scale=150:-1 %s", src, dest)
		assert.Equal(t, expected, string(actual))
	})

	t.Run("Original from video", func(t *testing.T) {
		t.Setenv("TEST_BEHAVIOR", "dump_cmd")
		converter := &FFmpegConverter{
			exe: testExe(t),
		}

		mediasDir := filepath.Join("testdata", "TestMedias/medias")
		outputDir := t.TempDir()

		src := filepath.Join(mediasDir, "forest-large.webm")
		dest := filepath.Join(outputDir, "out.avif")

		err := converter.ToAVIF(src, dest, OriginalSize())
		require.NoError(t, err)

		// Check cmd
		actual, err := os.ReadFile(dest)
		require.NoError(t, err)
		expected := fmt.Sprintf("ffmpeg -i %s -vf select=eq(n\\,0) %s", src, dest)
		assert.Equal(t, expected, string(actual))
	})

}

func TestConvertToMP3(t *testing.T) {

	t.Run("Basic", func(t *testing.T) {
		t.Setenv("TEST_BEHAVIOR", "dump_cmd")
		converter := &FFmpegConverter{
			exe: testExe(t),
		}

		mediasDir := filepath.Join("testdata", "TestMedias/medias")
		outputDir := t.TempDir()

		src := filepath.Join(mediasDir, "waterfall.flac")
		dest := filepath.Join(outputDir, "out.mp3")

		err := converter.ToMP3(src, dest)
		require.NoError(t, err)

		// Check cmd
		actual, err := os.ReadFile(dest)
		require.NoError(t, err)
		expected := fmt.Sprintf("ffmpeg -i %s %s", src, dest)
		assert.Equal(t, expected, string(actual))
	})

}

func TestConvertToWebM(t *testing.T) {

	t.Run("Basic", func(t *testing.T) {
		t.Setenv("TEST_BEHAVIOR", "dump_cmd")
		converter := &FFmpegConverter{
			exe: testExe(t),
		}

		mediasDir := filepath.Join("testdata", "TestMedias/medias")
		outputDir := t.TempDir()

		src := filepath.Join(mediasDir, "aurora.mp4")
		dest := filepath.Join(outputDir, "out.webm")

		err := converter.ToWebM(src, dest)
		require.NoError(t, err)

		// Check cmd
		actual, err := os.ReadFile(dest)
		require.NoError(t, err)
		expected := fmt.Sprintf("ffmpeg -i %s %s", src, dest)
		assert.Equal(t, expected, string(actual))
	})

}

func TestReadImageDimensions(t *testing.T) {
	mediasDir := filepath.Join("testdata", "TestMedias/medias")

	tests := []struct {
		filename     string // input
		width        int    // output
		height       int    // output
		errorMessage string // output
	}{
		{"tree-landscape-large.jpg", 2400, 1800, ""},
		{"bird-landscape-medium.png", 1280, 891, ""},
		{"tree-landscape.avif", 0, 0, "unknown format"},
		{"earth-landscape-large.gif", 400, 254, ""},
	}
	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			src := filepath.Join(mediasDir, tt.filename)
			dimensions, err := ReadImageDimensions(src)
			if tt.errorMessage != "" {
				require.ErrorContains(t, err, tt.errorMessage)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.width, dimensions.Width)
				assert.Equal(t, tt.height, dimensions.Height)
			}
		})
	}
}

/* Test Helpers */

func testExe(t *testing.T) string {
	// The trick is to override the command name to inject the go test binary.
	// Tests define the environment variable TEST_GIT_BEHAVIOR to determine
	// the behavior of the replaced command.
	testExe, err := os.Executable()
	require.NoError(t, err, "can't determine current exectuable")
	return testExe
}
