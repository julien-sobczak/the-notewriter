package medias

import (
	"fmt"
	"image"
	"os"
)

// Dimensions regroups the width and height of an image.
type Dimensions struct {
	Width  int
	Height int
}

func ResizeTo(maxWidthOrHeight int) Dimensions {
	return Dimensions{
		Width:  maxWidthOrHeight,
		Height: maxWidthOrHeight,
	}
}

func OriginalSize() Dimensions {
	return Dimensions{
		Width:  0,
		Height: 0,
	}
}

// Zero returns if the dimensions are not available.
func (d Dimensions) Zero() bool {
	return d.Height == 0 && d.Width == 0
}

func (d Dimensions) Landscape() bool {
	if d.Zero() {
		return false
	}
	return d.Width > d.Height
}

func (d Dimensions) Portrait() bool {
	if d.Zero() {
		return false
	}
	return d.Width < d.Height
}

// LargerThan returns if at least one dimension exceeds the given size.
// LargerThan is conservatrice and returns true if dimensions are not available.
func (d Dimensions) LargerThan(widthOrHeight int) bool {
	if d.Zero() {
		// Considers any image to be larger if dimensions are not available
		// Upscaling will be performed...
		return true
	}
	return d.Height > widthOrHeight || d.Width > widthOrHeight
}

func (d Dimensions) String() string {
	return fmt.Sprintf("%dx%d", d.Width, d.Height)
}

// ReadImageDimensions extracts the dimensions from a GIF/PNG/JPEG file.
func ReadImageDimensions(path string) (Dimensions, error) {
	f, err := os.Open(path)
	if err != nil {
		return Dimensions{}, err
	}
	config, _, err := image.DecodeConfig(f)
	if err != nil {
		return Dimensions{}, err
	}
	return Dimensions{
		Width:  config.Width,
		Height: config.Height,
	}, nil
}

type Converter interface {
	OnPreGeneration(func(cmd string, args ...string))
	ToAVIF(src, dest string, dimensions Dimensions) error
	ToMP3(src, dest string) error
	ToWebM(src, dest string) error
}
