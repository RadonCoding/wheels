package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"io"
	"log"
	"math"
	"runtime"
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

func drawWheelBorderImage(width, height int, cx, cy, radius float64) image.Image {
	dc := gg.NewContext(width, height)

	outerRadius := radius
	innerRadius := radius * 0.95

	dc.SetColor(colorOutline)

	dc.NewSubPath()
	dc.DrawCircle(cx, cy, outerRadius)

	dc.NewSubPath()
	dc.DrawCircle(cx, cy, innerRadius)
	dc.SetFillRule(gg.FillRuleEvenOdd)
	dc.Fill()

	return dc.Image()
}

const (
	BLINKING_START = 0.9
	BLINKING_END   = 1.0
)

func getTextColor(animation float64) color.Color {
	if animation < BLINKING_START {
		return color.White
	}
	if animation <= BLINKING_END {
		phase := (animation - BLINKING_START) / (BLINKING_END - BLINKING_START)
		return interpolate(color.White, colorRed, phase)
	}
	return colorRed
}

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

func drawWheel(dc *gg.Context, options []string, target int, cx, cy, radius, animation, rotation float64) {
	outerRadius := radius
	innerRadius := outerRadius * 0.95

	// Load better font
	regular, err := truetype.Parse(goregular.TTF)
	if err != nil {
		log.Fatal(err)
	}
	dc.SetFontFace(truetype.NewFace(regular, &truetype.Options{
		Size:    radius * 0.06,
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

		// Draw the label
		dc.SetColor(color.Black)
		dc.DrawStringAnchored(label, 1, 1, 0.5, 0.5)
		var fg color.Color = color.White
		if i == target {
			fg = getTextColor(animation)
		}
		dc.SetColor(fg)
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

func drawLights(dc *gg.Context, options []string, cx, cy, radius, animation, rotation float64) {
	outerRadius := radius
	innerRadius := outerRadius * 0.95

	lightCount := len(options) * 2
	lightRadius := radius * 0.015
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

func drawArrow(dc *gg.Context, cx, cy, radius, animation float64) {
	outerRadius := radius
	innerRadius := outerRadius * 0.95

	arrowSize := radius * 0.15
	// Places the arrow slightly inside the border
	arrowOffset := innerRadius + arrowSize*0.1

	tipX := cx
	tipY := cy - arrowOffset

	dc.NewSubPath()
	dc.MoveTo(tipX, tipY+arrowSize)
	dc.LineTo(tipX-arrowSize/2, tipY)
	dc.LineTo(tipX+arrowSize/2, tipY)
	dc.ClosePath()

	animated := getArrowColor(animation)
	dc.SetColor(animated)
	dc.FillPreserve()

	scaledLineWidth := radius * 0.01
	dc.SetLineWidth(scaledLineWidth)
	dc.SetColor(colorOutline)
	dc.Stroke()
}

func generateWheelGIF(w io.Writer, options []string, target, fps, duration int) error {
	const RADIUS = 200

	const W, H = RADIUS * 2, RADIUS * 2

	cx, cy := float64(W)/2, float64(H)/2

	spins := float64(duration) * 1.0
	rotations := spins * 2 * math.Pi
	required := clockWiseToTarget(options, target)
	destination := rotations + required

	delay := 100 / fps

	frames := fps * duration

	images := make([]*image.Paletted, frames)
	delays := make([]int, frames)

	var wg sync.WaitGroup
	wg.Add(frames)

	workerCount := runtime.NumCPU()
	sem := make(chan struct{}, workerCount)

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

	border := drawWheelBorderImage(W, H, cx, cy, RADIUS)

	// Processing frames in parallel :>
	for frame := 0; frame < frames; frame++ {
		sem <- struct{}{}
		go func(frame int) {
			defer func() { <-sem }()
			defer wg.Done()

			animation := float64(frame) / float64(frames-1)
			eased := 1 - math.Pow(1-animation, 3)

			frameDC := gg.NewContext(W, H)

			rotation := math.Min(destination, destination*eased)
			drawWheel(frameDC, options, target, cx, cy, RADIUS, animation, rotation)
			drawArrow(frameDC, cx, cy, RADIUS, animation)
			frameDC.DrawImage(border, 0, 0)
			drawLights(frameDC, options, cx, cy, RADIUS, animation, rotation)

			render := frameDC.Image()
			bounds := render.Bounds()
			paletted := image.NewPaletted(bounds, palette)
			draw.Draw(paletted, bounds, render, bounds.Min, draw.Src)
			images[frame] = paletted
			delays[frame] = delay
		}(frame)
	}

	wg.Wait()

	return gif.EncodeAll(w, &gif.GIF{
		Image:     images,
		Delay:     delays,
		LoopCount: -1,
	})
}
