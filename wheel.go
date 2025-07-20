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

type Theme struct {
	Transparent color.Color // #00000000
	Black       color.Color // #000000
	White       color.Color // #FFFFFF

	BgPrimary      color.Color // #36393F
	BgSecondary    color.Color // #2F3136
	Outline        color.Color // #202225
	Blurple        color.Color // #5865F2
	Red            color.Color // #ED4245
	ArrowInactive  color.Color // #B9BBBE
	LightsInactive color.Color // #4F545C

	Font    *truetype.Font
	Palette []color.Color
}

type WheelRenderer struct {
	OuterRadius float64
	InnerRadius float64
	Options     []string
	Target      int
	FPS         int
	Duration    int
}

func NewDefaultTheme() Theme {
	ttf, err := truetype.Parse(goregular.TTF)
	if err != nil {
		log.Fatal(err)
	}

	theme := Theme{
		Transparent:    color.Transparent,              // #00000000
		Black:          color.Black,                    // #000000
		White:          color.White,                    // #FFFFFF
		BgPrimary:      color.RGBA{54, 57, 63, 255},    // #36393F
		BgSecondary:    color.RGBA{47, 49, 54, 255},    // #2F3136
		Outline:        color.RGBA{32, 34, 37, 255},    // #202225
		Blurple:        color.RGBA{88, 101, 242, 255},  // #5865F2
		Red:            color.RGBA{237, 66, 69, 255},   // #ED4245
		ArrowInactive:  color.RGBA{185, 187, 190, 255}, // #B9BBBE
		LightsInactive: color.RGBA{79, 84, 92, 255},    // #4F545C
		Font:           ttf,
	}

	theme.Palette = []color.Color{
		theme.Transparent,
		theme.Black,
		theme.White,
		theme.BgPrimary,
		theme.BgSecondary,
		theme.Outline,
		theme.Blurple,
		theme.Red,
		theme.ArrowInactive,
		theme.LightsInactive,
	}

	return theme
}

func (wr *WheelRenderer) drawWheelBorderImage(theme Theme, width, height int, cx, cy float64) image.Image {
	dc := gg.NewContext(width, height)

	dc.SetColor(theme.Outline)

	dc.NewSubPath()
	dc.DrawCircle(cx, cy, wr.OuterRadius)

	dc.NewSubPath()
	dc.DrawCircle(cx, cy, wr.InnerRadius)
	dc.SetFillRule(gg.FillRuleEvenOdd)
	dc.Fill()

	return dc.Image()
}

const (
	BLINKING_START = 0.9
	BLINKING_END   = 1.0
)

func getTextColor(theme Theme, animation float64) color.Color {
	if animation < BLINKING_START {
		return theme.White
	}
	if animation <= BLINKING_END {
		phase := (animation - BLINKING_START) / (BLINKING_END - BLINKING_START)
		return interpolate(theme.White, theme.Red, phase)
	}
	return theme.Red
}

func getLightColor(theme Theme, angle, animation, rotation float64) color.Color {
	if animation < BLINKING_START {
		relative := math.Mod(angle-rotation, 2*math.Pi)
		progress := relative / (2 * math.Pi)
		brightness := (math.Sin(progress*2*math.Pi*3) + 1) / 2
		return interpolate(theme.LightsInactive, theme.Blurple, brightness)
	}
	if animation >= BLINKING_START && animation <= BLINKING_END {
		phase := (animation - BLINKING_START) / (BLINKING_END - BLINKING_START)
		return interpolate(theme.LightsInactive, theme.Blurple, phase)
	}
	return theme.Red
}

func getArrowColor(theme Theme, animation float64) color.Color {
	if animation < BLINKING_START {
		return theme.ArrowInactive
	}
	if animation >= BLINKING_START && animation <= BLINKING_END {
		phase := (animation - BLINKING_START) / (BLINKING_END - BLINKING_START)
		return interpolate(theme.ArrowInactive, theme.Red, phase)
	}
	return theme.Red
}

func (wr *WheelRenderer) drawWheel(theme Theme, dc *gg.Context, cx, cy, animation, rotation float64) {
	// Draw wheel
	angle := 2 * math.Pi / float64(len(wr.Options))

	for i, label := range wr.Options {
		start := rotation + angle*float64(i)
		end := start + angle

		dc.MoveTo(cx, cy)
		dc.DrawArc(cx, cy, wr.InnerRadius, start, end)
		dc.ClosePath()

		if i%2 == 0 {
			dc.SetColor(theme.BgPrimary)
		} else {
			dc.SetColor(theme.BgSecondary)
		}
		dc.Fill()

		center := (start + end) / 2

		labelX := cx + math.Cos(center)*wr.InnerRadius*0.6
		labelY := cy + math.Sin(center)*wr.InnerRadius*0.6

		dc.Push()
		dc.Translate(labelX, labelY)

		// Draw the label
		dc.SetColor(theme.Black)
		dc.DrawStringAnchored(label, 1, 1, 0.5, 0.5)

		var fg color.Color = theme.White
		if i == wr.Target {
			fg = getTextColor(theme, animation)
		}
		dc.SetColor(fg)
		dc.DrawStringAnchored(label, 0, 0, 0.5, 0.5)

		dc.Pop()
	}

	// Draw division lines
	dc.SetLineWidth(2)
	dc.SetColor(theme.Outline)
	for i := 0; i < len(wr.Options); i++ {
		angle := rotation + angle*float64(i)
		x := cx + math.Cos(angle)*wr.InnerRadius
		y := cy + math.Sin(angle)*wr.InnerRadius
		dc.MoveTo(cx, cy)
		dc.LineTo(x, y)
		dc.Stroke()
	}

	// Draw hub
	hubRadius := wr.OuterRadius * 0.20
	dc.SetColor(theme.Blurple)
	dc.DrawCircle(cx, cy, hubRadius)
	dc.Fill()
	dc.SetLineWidth(2)
	dc.SetColor(theme.Outline)
	dc.DrawCircle(cx, cy, hubRadius)
	dc.Stroke()
}

func (wr *WheelRenderer) drawLights(theme Theme, dc *gg.Context, cx, cy, animation, rotation float64) {
	count := len(wr.Options) * 2
	radius := wr.OuterRadius * 0.015
	offset := (wr.OuterRadius + wr.InnerRadius) / 2
	step := 2 * math.Pi / float64(count)

	for i := 0; i < count; i++ {
		angle := step * float64(i)
		x := cx + math.Cos(angle)*offset
		y := cy + math.Sin(angle)*offset

		animated := getLightColor(theme, angle, animation, rotation)

		dc.SetColor(animated)
		dc.DrawCircle(x, y, radius)
		dc.Fill()
	}
}

func (wr *WheelRenderer) drawArrow(theme Theme, dc *gg.Context, cx, cy, animation float64) {
	size := wr.OuterRadius * 0.15
	// Places the arrow slightly inside the border
	offset := wr.InnerRadius + size*0.1

	tipX := cx
	tipY := cy - offset

	dc.NewSubPath()
	dc.MoveTo(tipX, tipY+size)
	dc.LineTo(tipX-size/2, tipY)
	dc.LineTo(tipX+size/2, tipY)
	dc.ClosePath()

	animated := getArrowColor(theme, animation)
	dc.SetColor(animated)
	dc.FillPreserve()

	scaledLineWidth := wr.OuterRadius * 0.01
	dc.SetLineWidth(scaledLineWidth)
	dc.SetColor(theme.Outline)
	dc.Stroke()
}

func distanceToTarget(count, index int) float64 {
	angle := 2 * math.Pi / float64(count)
	start := angle * float64(index)
	end := start + angle
	center := (start + end) / 2
	const top = (3 * math.Pi / 2)
	return math.Mod(top-center+2*math.Pi, 2*math.Pi)
}

func (wr *WheelRenderer) RenderGIF(w io.Writer) error {
	width, height := int(wr.OuterRadius*2), int(wr.OuterRadius*2)
	cx, cy := float64(width)/2, float64(height)/2

	spins := float64(wr.Duration) * 1.0
	circles := spins * 2 * math.Pi
	required := distanceToTarget(len(wr.Options), wr.Target)
	destination := circles + required

	delay := 100 / wr.FPS

	frames := wr.FPS * wr.Duration

	images := make([]*image.Paletted, frames)
	delays := make([]int, frames)

	var wg sync.WaitGroup
	wg.Add(frames)

	sem := make(chan struct{}, runtime.NumCPU())

	theme := NewDefaultTheme()

	border := wr.drawWheelBorderImage(theme, width, height, cx, cy)

	// Processing frames in parallel :>
	for frame := 0; frame < frames; frame++ {
		sem <- struct{}{}
		go func(frame int) {
			defer func() { <-sem }()
			defer wg.Done()

			animation := float64(frame) / float64(frames-1)
			eased := 1 - math.Pow(1-animation, 3)

			rotation := math.Min(destination, destination*eased)

			dc := gg.NewContext(width, height)
			dc.SetColor(theme.Transparent)
			dc.Clear()

			dc.SetFontFace(truetype.NewFace(theme.Font, &truetype.Options{
				Size:    wr.OuterRadius * 0.05,
				Hinting: font.HintingFull,
			}))

			wr.drawWheel(theme, dc, cx, cy, animation, rotation)
			wr.drawArrow(theme, dc, cx, cy, animation)
			dc.DrawImage(border, 0, 0)
			wr.drawLights(theme, dc, cx, cy, animation, rotation)

			render := dc.Image()
			bounds := render.Bounds()
			paletted := image.NewPaletted(bounds, theme.Palette)
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
