package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"io"
	"log"
	"math"
	"sync"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"github.com/lucasb-eyer/go-colorful"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
)

func drawWheel(dc *gg.Context, options []string, cx float64, cy float64, radius float64, rotation float64) {
	outerRadius := radius
	innerRadius := radius * 0.9

	// Draw outline
	dc.SetRGB255(50, 50, 50)
	dc.DrawCircle(cx, cy, outerRadius)
	dc.Fill()
	dc.SetRGBA(0, 0, 0, 0)
	dc.DrawCircle(cx, cy, innerRadius)
	dc.Fill()

	// Load better font
	regular, err := truetype.Parse(goregular.TTF)
	if err != nil {
		log.Fatal(err)
	}
	dc.SetFontFace(truetype.NewFace(regular, &truetype.Options{
		Size:    14,
		Hinting: font.HintingFull,
	}))

	// Draw wheel
	angleStep := 2 * math.Pi / float64(len(options))

	for i, label := range options {
		startAngle := rotation + angleStep*float64(i)
		endAngle := startAngle + angleStep

		dc.MoveTo(cx, cy)
		dc.DrawArc(cx, cy, innerRadius, startAngle, endAngle)

		hue := float64(i) / float64(len(options)) * 360.0
		bg := colorful.Hsv(hue, 0.8, 0.9)
		dc.SetColor(bg)
		dc.Fill()

		midAngle := (startAngle + endAngle) / 2
		labelX := cx + math.Cos(midAngle)*innerRadius*0.6
		labelY := cy + math.Sin(midAngle)*innerRadius*0.6

		dc.Push()
		dc.Translate(labelX, labelY)
		dc.Rotate(midAngle)

		angleDeg := midAngle * 180 / math.Pi
		if angleDeg > 90 && angleDeg < 270 {
			dc.Rotate(math.Pi)
		}

		dc.SetColor(color.Black)
		dc.DrawStringAnchored(label, 0, 0, 0.5, 0.5)

		dc.Pop()
	}

	// Draw division lines
	dc.SetLineWidth(4)
	dc.SetRGB255(50, 50, 50)
	for i := 0; i < len(options); i++ {
		angle := rotation + angleStep*float64(i)
		x := cx + math.Cos(angle)*innerRadius
		y := cy + math.Sin(angle)*innerRadius
		dc.MoveTo(cx, cy)
		dc.LineTo(x, y)
		dc.Stroke()
	}
}

const (
	BLINKING_START = 0.9
	BLINKING_END   = 1.8
	FADE_OUT_TIME  = 0.2
	FADE_OUT_END   = BLINKING_END + FADE_OUT_TIME
)

func getLightColor(angle float64, animation, rotation float64, inactive, spinning, stopped color.RGBA) color.RGBA {
	if animation < BLINKING_START {
		relative := math.Mod(angle-rotation, 2*math.Pi)
		progress := relative / (2 * math.Pi)
		brightness := (math.Sin(progress*2*math.Pi*3) + 1) / 2
		return interpolate(inactive, spinning, brightness)
	}

	if animation >= BLINKING_START && animation <= FADE_OUT_END {
		if animation <= BLINKING_END {
			normalized := (animation - BLINKING_START) / (BLINKING_END - BLINKING_START)
			blink := (math.Sin(normalized*20.0*math.Pi) + 1) / 2
			return interpolate(stopped, inactive, blink)
		}
		fade := (animation - BLINKING_END) / FADE_OUT_TIME
		return interpolate(stopped, inactive, fade)
	}

	return inactive
}

func getArrowColor(animation float64, inactive, stopped color.RGBA) color.RGBA {
	if animation >= BLINKING_START && animation <= FADE_OUT_END {
		if animation <= BLINKING_END {
			normalized := (animation - BLINKING_START) / (BLINKING_END - BLINKING_START)
			blink := (math.Sin(normalized*20.0*math.Pi) + 1) / 2
			return interpolate(stopped, inactive, blink)
		}
		fade := (animation - BLINKING_END) / FADE_OUT_TIME
		return interpolate(stopped, inactive, fade)
	}

	return inactive
}

func drawLights(dc *gg.Context, options []string, cx float64, cy float64, radius float64, animation float64, rotation float64) {
	outerRadius := radius
	innerRadius := radius * 0.9

	inactive := color.RGBA{R: 100, G: 100, B: 100, A: 255}
	spinning := color.RGBA{R: 255, G: 255, B: 0, A: 255}
	stopped := color.RGBA{R: 255, G: 0, B: 0, A: 255}

	// Draw lights
	lightCount := len(options) * 3
	lightRadius := float64(lightCount) / 4
	lightOffset := (outerRadius + innerRadius) / 2
	angleStep := 2 * math.Pi / float64(lightCount)

	for i := 0; i < lightCount; i++ {
		angle := angleStep * float64(i)
		x := cx + math.Cos(angle)*lightOffset
		y := cy + math.Sin(angle)*lightOffset

		animated := getLightColor(angle, animation, rotation, inactive, spinning, stopped)

		dc.SetRGBA255(int(animated.R), int(animated.G), int(animated.B), int(animated.A))
		dc.DrawCircle(x, y, lightRadius)
		dc.Fill()
	}
}

func drawArrow(dc *gg.Context, cx float64, cy float64, radius float64, animation float64) {
	innerRadius := radius * 0.9

	inactive := color.RGBA{R: 100, G: 100, B: 100, A: 255}
	stopped := color.RGBA{R: 255, G: 0, B: 0, A: 255}

	const ARROW_SIZE = 32.0

	tipX := cx
	tipY := cy - innerRadius + ARROW_SIZE

	// Draw arrow
	dc.NewSubPath()
	dc.MoveTo(tipX, tipY)
	dc.LineTo(tipX-ARROW_SIZE/2, tipY-ARROW_SIZE)
	dc.LineTo(tipX+ARROW_SIZE/2, tipY-ARROW_SIZE)
	dc.ClosePath()

	// Fill arrow color
	animated := getArrowColor(animation, inactive, stopped)
	dc.SetRGBA255(int(animated.R), int(animated.G), int(animated.B), int(animated.A))
	dc.FillPreserve()

	// Draw outline
	dc.SetLineWidth(3)
	dc.SetRGB(0, 0, 0)
	dc.Stroke()
}

// First draws the wheel, then the arrow and rotates the drawn wheel image while layering the lights and arrow on top for each frame
func generateWheelGIF(w io.Writer, options []string, target int, fps int, spins int, duration int, linger int) error {
	const RADIUS = 200

	const W, H = RADIUS * 2, RADIUS * 2

	cx, cy := float64(W)/2, float64(H)/2

	destination := float64(spins)*2*math.Pi + clockWiseToTarget(options, target)

	delay := 100 / fps

	spinning := fps * duration
	lingering := fps * linger
	frames := spinning + lingering

	images := make([]*image.Paletted, frames)
	delays := make([]int, frames)

	rendered := make([]image.Image, frames)

	var wg sync.WaitGroup
	wg.Add(frames)

	for frame := 0; frame < frames; frame++ {
		go func(frame int) {
			defer wg.Done()

			animation := float64(frame) / float64(spinning-1)
			eased := 1 - math.Pow(1-animation, 3)

			dc := gg.NewContext(W, H)

			rotation := math.Min(destination, destination*eased)
			drawWheel(dc, options, cx, cy, RADIUS, rotation)

			drawLights(dc, options, cx, cy, RADIUS, animation, rotation)
			drawArrow(dc, cx, cy, RADIUS, animation)

			rendered[frame] = dc.Image()
		}(frame)
	}

	wg.Wait()

	combined := image.NewRGBA(image.Rect(0, 0, W, H*frames))
	for i, img := range rendered {
		draw.Draw(combined, image.Rect(0, i*H, W, (i+1)*H), img, image.Point{0, 0}, draw.Src)
	}
	palette := createPalette(combined, 128)

	for i, render := range rendered {
		bounds := render.Bounds()
		paletted := image.NewPaletted(bounds, palette)
		draw.Draw(paletted, bounds, render, bounds.Min, draw.Src)
		images[i] = paletted
		delays[i] = delay
	}

	return gif.EncodeAll(w, &gif.GIF{
		Image: images,
		Delay: delays,
	})
}
