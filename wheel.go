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
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
)

var (
	colorBgPrimary      = color.RGBA{54, 57, 63, 255}    // #36393F
	colorBgSecondary    = color.RGBA{47, 49, 54, 255}    // #2F3136
	colorOutline        = color.RGBA{32, 34, 37, 255}    // #202225
	colorBlurple        = color.RGBA{88, 101, 242, 255}  // #5865F2
	colorRed            = color.RGBA{237, 66, 69, 255}   // #ED4245
	colorArrowInactive  = color.RGBA{185, 187, 190, 255} // #B9BBBE
	colorLightsInactive = color.RGBA{79, 84, 92, 255}    // #4F545C
)

func drawWheel(dc *gg.Context, options []string, cx float64, cy float64, radius float64, rotation float64) {
	outerRadius := radius
	innerRadius := radius * 0.95 // Thinner border

	// Draw outline
	dc.SetColor(colorOutline)
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
		Size:    12,
		Hinting: font.HintingFull,
	}))

	// Draw wheel
	angleStep := 2 * math.Pi / float64(len(options))

	for i, label := range options {
		startAngle := rotation + angleStep*float64(i)
		endAngle := startAngle + angleStep

		dc.MoveTo(cx, cy)
		dc.DrawArc(cx, cy, innerRadius, startAngle, endAngle)
		dc.ClosePath()

		if i%2 == 0 {
			dc.SetColor(colorBgPrimary)
		} else {
			dc.SetColor(colorBgSecondary)
		}
		dc.Fill()

		midAngle := (startAngle + endAngle) / 2
		labelX := cx + math.Cos(midAngle)*innerRadius*0.6
		labelY := cy + math.Sin(midAngle)*innerRadius*0.6

		dc.Push()
		dc.Translate(labelX, labelY)

		if midAngle > math.Pi/2 && midAngle < 3*math.Pi/2 {
			dc.Rotate(math.Pi)
		}

		// Draw the label
		dc.SetColor(color.Black)
		dc.DrawStringAnchored(label, 1, 1, 0.5, 0.5)
		dc.SetColor(color.White)
		dc.DrawStringAnchored(label, 0, 0, 0.5, 0.5)

		dc.Pop()
	}

	// Draw division lines
	dc.SetLineWidth(2)
	dc.SetColor(colorOutline)
	for i := 0; i < len(options); i++ {
		angle := rotation + angleStep*float64(i)
		x := cx + math.Cos(angle)*innerRadius
		y := cy + math.Sin(angle)*innerRadius
		dc.MoveTo(cx, cy)
		dc.LineTo(x, y)
		dc.Stroke()
	}

	// Draw hub
	hubRadius := radius * 0.20
	dc.SetColor(colorBlurple)
	dc.DrawCircle(cx, cy, hubRadius)
	dc.Fill()
	dc.SetLineWidth(2)
	dc.SetColor(colorOutline)
	dc.DrawCircle(cx, cy, hubRadius)
	dc.Stroke()
}

const (
	BLINKING_START = 0.9
	BLINKING_END   = 1.0
)

func getLightColor(angle, animation, rotation float64) color.RGBA {
	if animation < BLINKING_START {
		relative := math.Mod(angle-rotation, 2*math.Pi)
		progress := relative / (2 * math.Pi)
		brightness := (math.Sin(progress*2*math.Pi*3) + 1) / 2
		return interpolate(colorLightsInactive, colorBlurple, brightness)
	}
	if animation >= BLINKING_START && animation <= BLINKING_END {
		phase := (animation - BLINKING_START) / (BLINKING_END - BLINKING_START)
		return interpolate(colorLightsInactive, colorBlurple, phase)
	}
	return colorRed
}

func getArrowColor(animation float64) color.RGBA {
	if animation < BLINKING_START {
		return colorArrowInactive
	}
	if animation >= BLINKING_START && animation <= BLINKING_END {
		phase := (animation - BLINKING_START) / (BLINKING_END - BLINKING_START)
		return interpolate(colorArrowInactive, colorRed, phase)
	}
	return colorRed
}

func drawLights(dc *gg.Context, options []string, cx float64, cy float64, radius float64, animation float64, rotation float64) {
	outerRadius := radius
	innerRadius := radius * 0.95

	// Draw lights
	lightCount := len(options) * 2
	lightRadius := 3.0
	lightOffset := (outerRadius + innerRadius) / 2
	angleStep := 2 * math.Pi / float64(lightCount)

	for i := 0; i < lightCount; i++ {
		angle := angleStep * float64(i)
		x := cx + math.Cos(angle)*lightOffset
		y := cy + math.Sin(angle)*lightOffset

		animated := getLightColor(angle, animation, rotation)

		dc.SetColor(animated)
		dc.DrawCircle(x, y, lightRadius)
		dc.Fill()
	}
}

func drawArrow(dc *gg.Context, cx float64, cy float64, radius float64, animation float64) {
	outerRadius := radius

	const ARROW_SIZE = 30.0

	tipX := cx
	tipY := cy - outerRadius - 5

	// Draw arrow shape
	dc.NewSubPath()
	dc.MoveTo(tipX, tipY+ARROW_SIZE)
	dc.LineTo(tipX-ARROW_SIZE/2, tipY)
	dc.LineTo(tipX+ARROW_SIZE/2, tipY)
	dc.ClosePath()

	// Fill arrow color
	animated := getArrowColor(animation)
	dc.SetColor(animated)
	dc.FillPreserve()

	// Draw arrow outline
	dc.SetLineWidth(2)
	dc.SetColor(colorOutline)
	dc.Stroke()
}

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

			if frame >= spinning {
				animation = BLINKING_END
				eased = 1.0
			}

			dc := gg.NewContext(W, H)

			rotation := math.Min(destination, destination*eased)
			drawWheel(dc, options, cx, cy, RADIUS, rotation)

			drawLights(dc, options, cx, cy, RADIUS, animation, rotation)
			drawArrow(dc, cx, cy, RADIUS, animation)

			rendered[frame] = dc.Image()
		}(frame)
	}

	wg.Wait()

	palette := []color.Color{
		color.Transparent,
		color.Black,
		color.White,
		colorBgPrimary,
		colorBgSecondary,
		colorOutline,
		colorBlurple,
		colorRed,
		colorArrowInactive,
		colorLightsInactive,
	}

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
