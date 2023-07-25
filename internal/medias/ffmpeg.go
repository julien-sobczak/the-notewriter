package medias

import (
	"context"
	"errors"
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type FFmpegConverter struct {
	exe       string
	preset    string // ultrafast, superfast, veryfast, fast, medium, slow, slower, veryslow
	listeners []func(cmd string, args ...string)
}

func NewFFmpegConverter(preset string) (*FFmpegConverter, error) {
	path, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, errors.New("executable 'ffmpeg' not found in $PATH")
	}

	if preset == "" {
		preset = "medium"
	}

	return &FFmpegConverter{
		exe:    path,
		preset: preset,
	}, nil
}

func (c *FFmpegConverter) OnPreGeneration(fn func(cmd string, args ...string)) {
	c.listeners = append(c.listeners, fn)
}

func (c *FFmpegConverter) notifyListeners(cmd string, args ...string) {
	for _, fn := range c.listeners {
		fn(cmd, args...)
	}
}

/*
 * We use the tool ffmeg to convert between medias.
 * It's a huge external dependency but it's the most popular one
 * and the huge list of supported formats is convenient
 * as we just have to use a single command to process all kinds of medias.
 *
 * Here are a few example of its usage.
 *
 *
 * Extract the first frame of a video (-vf "select=eq(n\,0)")
 *    $ ffmpeg -i earth-landscape-large.gif -vf "select=eq(n\,0)" out-earth.avif
 *    $ ffmpeg -i <input> -vframes 1 <output>.jpeg
 *
 * Configure the output quality for JPEG (-q:v)
 *    $ ffmpeg -i inputfile.mkv -vf "select=eq(n\,0)" -q:v 3 output_image.jpg
 *
 * Scaling (see https://trac.ffmpeg.org/wiki/Scaling)
 *    $ ffmpeg -i input.jpg -vf scale=320:240 output_320x240.png
 *
 * Keeping the aspect ratio
 *    $ ffmpeg -i input.jpg -vf scale=320:-1 output_320.png
 *    $ ffmpeg -i input.jpg -vf scale=320:-2 output_320.png (on some codecs)
 */

// ConvertToAVIF convert a picture to AVIF format.
// Requirements:
//
//	brew install ffmpeg
func (c *FFmpegConverter) ToAVIF(srcPath string, destPath string, dimensions Dimensions) error {
	// Check dest extension
	destExt := strings.ToLower(filepath.Ext(destPath))
	if destExt != ".avif" {
		return fmt.Errorf("target file must used extension .avif. Go: %s", destExt)
	}

	// Check src file exists
	_, err := os.Stat(srcPath)
	if err != nil {
		return err
	}

	var cmdArgs []string
	var cmdFilters []string

	// Export the first frame for videos
	if strings.Contains(MimeType(filepath.Ext(srcPath)), "video") {
		// Export the first frame only
		cmdFilters = append(cmdFilters, `select=eq(n\,0)`)
	}

	// Apply scaling if required
	if !dimensions.Zero() {
		// Read dimensions to detect portrait/landscape
		srcDimensions, _ := ReadImageDimensions(srcPath)
		if srcDimensions.Portrait() {
			cmdFilters = append(cmdFilters, fmt.Sprintf("scale=-1:%d", dimensions.Height))
		} else {
			cmdFilters = append(cmdFilters, fmt.Sprintf("scale=%d:-1", dimensions.Width))
		}
	}

	var filtersArgs []string
	if len(cmdFilters) > 0 {
		filtersArgs = append(filtersArgs, "-vf", strings.Join(cmdFilters, ","))
	}

	var args []string
	args = append(args, "-i", srcPath)
	args = append(args, "-preset", c.preset)
	args = append(args, cmdArgs...)
	args = append(args, filtersArgs...)
	args = append(args, destPath)

	c.notifyListeners(c.exe, args...)
	cmd := exec.CommandContext(context.Background(), c.exe, args...)

	// Dump output to troubleshoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", output)
	}

	return err
}

func (c *FFmpegConverter) ToMP3(srcPath string, destPath string) error {
	// Check dest extension
	destExt := strings.ToLower(filepath.Ext(destPath))
	if destExt != ".mp3" {
		return fmt.Errorf("target file must used extension .avif. Go: %s", destExt)
	}

	// Check src file exists
	_, err := os.Stat(srcPath)
	if err != nil {
		return err
	}

	c.notifyListeners(c.exe, "-i", srcPath, destPath)
	cmd := exec.CommandContext(context.Background(), c.exe, "-i", srcPath, "-preset", c.preset, destPath)

	return cmd.Run()
}

func (c *FFmpegConverter) ToWebM(srcPath string, destPath string) error {
	// Check dest extension
	destExt := strings.ToLower(filepath.Ext(destPath))
	if destExt != ".webm" {
		return fmt.Errorf("target file must used extension .avif. Go: %s", destExt)
	}

	// Check src file exists
	_, err := os.Stat(srcPath)
	if err != nil {
		return err
	}

	c.notifyListeners(c.exe, "-i", srcPath, destPath)
	cmd := exec.CommandContext(context.Background(), c.exe, "-i", srcPath, "-preset", c.preset, destPath)

	return cmd.Run()
}
