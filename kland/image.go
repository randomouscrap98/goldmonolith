package kland

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"io"
	"math"
	"strconv"
	"strings"

	_ "image/jpeg"
	_ "image/png"
	//"log"
)

const (
	AnimationWidth  = 200
	AnimationHeight = 100
	MaxFrameColors  = 4
)

type KlandAnimation struct {
	Version       int      `json:"version"`
	DefaultFrames string   `json:"defaultFrames"`
	Repeat        bool     `json:"repeat"`
	Times         []int    `json:"times"`
	Data          []string `json:"data"`
}

// Given a normal image data url (like image/png,base64), give back a reader which
// will give the raw bytes and the mimetype as provided by the data string
func ParseImageDataUrl(data string) (io.Reader, string, error) {
	firstComma := strings.IndexRune(data, ',')
	var mime string
	if firstComma >= 0 {
		mime = data[:firstComma]
		data = data[firstComma+1:]
	} else {
		return nil, "", fmt.Errorf("Bad image data url format (missing mime type)")
	}
	reader := strings.NewReader(data)
	return base64.NewDecoder(base64.StdEncoding, reader), mime, nil
}

// Scan through the image within the given bounds and compute the 4 color palette.
// Only useful for kland animations (use something else for other things)
func computeFramePalette(img image.Image, bounds image.Rectangle) []color.Color {
	uniqueColors := make(map[color.Color]struct{})
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			pixelColor := img.At(x, y)
			uniqueColors[pixelColor] = struct{}{}
		}
	}
	// Create a palette with the unique colors
	palette := make([]color.Color, MaxFrameColors)
	count := 0
	for col := range uniqueColors {
		palette[count] = col
		count += 1
		if count >= MaxFrameColors {
			break
		}
	}
	for i := count; i < MaxFrameColors; i++ {
		palette[i] = color.Black
	}
	return palette
}

// Figure out the palette index for this given color, or 0 if it doesn't exist
func computeColorIndex(c color.Color, palette []color.Color) uint8 {
	for i, col := range palette {
		if c == col {
			return uint8(i)
		}
	}
	return 0 // Just use the first color for any colors not found (is this ok???)
}

// Given a raw animation (which is probably json), write the converted gif to the
// given writer
func ConvertAnimation(rawAnimation string, outfile io.Writer) error {
	var anim KlandAnimation
	err := json.Unmarshal([]byte(rawAnimation), &anim)
	if err != nil {
		return err
	}
	defaultFrames, err := strconv.Atoi(anim.DefaultFrames)
	if err != nil {
		return err
	}
	gifframes := make([]*image.Paletted, len(anim.Data))
	delays := make([]int, len(anim.Data))
	for fi, framedata := range anim.Data {
		// -- Read image
		//log.Printf("Frame data:\n%s", framedata)
		reader, _, err := ParseImageDataUrl(framedata)
		if err != nil {
			return err
		}
		frameimg, _, err := image.Decode(reader)
		if err != nil {
			return err
		}
		// -- Compute (new) bounds
		bounds := frameimg.Bounds()
		if bounds.Dx() < AnimationWidth || bounds.Dy() < AnimationHeight {
			return fmt.Errorf("Bad dimensions on frame %d", fi)
		}
		bounds.Max.X = bounds.Min.X + 200
		bounds.Max.Y = bounds.Min.Y + 100
		// -- Compute delay
		delays[fi] = defaultFrames
		if anim.Times[fi] != 0 {
			delays[fi] = anim.Times[fi]
		}
		delays[fi] = int(math.Round(100.0 / 60.0 * float64(delays[fi])))
		// -- Create initial gif frame
		palette := computeFramePalette(frameimg, bounds)
		gifframe := image.NewPaletted(bounds, palette)
		// Very slowly copy the pixel data in by scanning for the right palette number
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				gifframe.SetColorIndex(x, y, computeColorIndex(frameimg.At(x, y), palette))
			}
		}

		gifframes[fi] = gifframe
	}

	options := gif.GIF{
		Image: gifframes,
		Delay: delays,
	}

	if anim.Repeat {
		options.LoopCount = 0
	} else {
		options.LoopCount = -1
	}

	return gif.EncodeAll(outfile, &options)
}
