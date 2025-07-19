package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/fogleman/gg"
	"github.com/joho/godotenv"
)

func hslToRgb(h, s, l float64) color.Color {
	c := (1 - math.Abs(2*l-1)) * s
	hp := h * 6
	x := c * (1 - math.Abs(math.Mod(hp, 2)-1))

	var r, g, b float64
	switch {
	case 0 <= hp && hp < 1:
		r, g, b = c, x, 0
	case 1 <= hp && hp < 2:
		r, g, b = x, c, 0
	case 2 <= hp && hp < 3:
		r, g, b = 0, c, x
	case 3 <= hp && hp < 4:
		r, g, b = 0, x, c
	case 4 <= hp && hp < 5:
		r, g, b = x, 0, c
	case 5 <= hp && hp < 6:
		r, g, b = c, 0, x
	default:
		r, g, b = 0, 0, 0
	}

	m := l - c/2
	r, g, b = r+m, g+m, b+m

	return color.RGBA{
		R: uint8(r * 255),
		G: uint8(g * 255),
		B: uint8(b * 255),
		A: 255,
	}
}

func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// Returns black or white depending on the color brightness
func getContrastColor(bg color.Color) color.Color {
	r, g, b, _ := bg.RGBA()
	brightness := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 65535
	if brightness > 0.5 {
		return color.Black
	}
	return color.White
}

func drawWheel(dc *gg.Context, options []string, cx float64, cy float64, radius float64) {
	// Draw wheel
	angleStep := 2 * math.Pi / float64(len(options))

	for i, label := range options {
		startAngle := angleStep * float64(i)
		endAngle := startAngle + angleStep

		dc.MoveTo(cx, cy)
		dc.DrawArc(cx, cy, radius, startAngle, endAngle)
		dc.ClosePath()

		hue := float64(i) / float64(len(options))
		color := hslToRgb(hue, 0.5, 0.5)
		dc.SetColor(color)
		dc.FillPreserve()

		dc.SetRGB(0, 0, 0)
		dc.Stroke()

		midAngle := (startAngle + endAngle) / 2
		labelX := cx + math.Cos(midAngle)*radius*0.5
		labelY := cy + math.Sin(midAngle)*radius*0.5

		dc.Push()
		dc.Translate(labelX, labelY)
		dc.Rotate(midAngle)

		angleDeg := midAngle * 180 / math.Pi
		if angleDeg > 90 && angleDeg < 270 {
			dc.Rotate(math.Pi)
		}

		contrast := getContrastColor(color)

		dc.SetColor(contrast)
		dc.DrawStringAnchored(label, 0, 0, 0.5, 0.5)
		dc.Pop()
	}

	// Draw outline
	dc.SetLineWidth(8)
	dc.SetRGB(0, 0, 0)
	dc.DrawCircle(cx, cy, radius)
	dc.Stroke()

	// Draw division lines
	dc.SetLineWidth(4)
	dc.SetRGB(0, 0, 0)
	for i := 0; i < len(options); i++ {
		angle := angleStep * float64(i)
		x := cx + math.Cos(angle)*radius
		y := cy + math.Sin(angle)*radius
		dc.MoveTo(cx, cy)
		dc.LineTo(x, y)
		dc.Stroke()
	}
}

func drawArrow(dc *gg.Context, cx float64, cy float64, radius float64) {
	const arrowSize = 40.0

	tipX := cx
	tipY := cy - radius + (arrowSize / 2)

	// Draw arrow
	dc.NewSubPath()
	dc.MoveTo(tipX, tipY)
	dc.LineTo(tipX-arrowSize/2, tipY-arrowSize)
	dc.LineTo(tipX+arrowSize/2, tipY-arrowSize)
	dc.ClosePath()

	// Fill arrow color
	dc.SetRGB(1, 0, 0)
	dc.FillPreserve()

	// Draw outline
	dc.SetLineWidth(3)
	dc.SetRGB(0, 0, 0)
	dc.Stroke()
}

// Uses the same logic as the wheel drawing to calculate how much to rotate to reach a specific option
func calculateRotationOffset(options []string, target int) float64 {
	angleStep := 2 * math.Pi / float64(len(options))
	startAngle := angleStep * float64(target)
	endAngle := startAngle + angleStep
	midAngle := (startAngle + endAngle) / 2
	return math.Mod((3*math.Pi/2)-midAngle+2*math.Pi, 2*math.Pi)
}

// Creates a palette by sampling pixels from an image
func createPalette(img image.Image, max int) []color.Color {
	colors := make(map[color.RGBA]struct{})
	bounds := img.Bounds()

	step := 4

	for y := bounds.Min.Y; y < bounds.Max.Y; y += step {
		for x := bounds.Min.X; x < bounds.Max.X; x += step {
			r, g, b, a := img.At(x, y).RGBA()
			c := color.RGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: uint8(a >> 8),
			}
			colors[c] = struct{}{}

			if len(colors) >= max {
				break
			}
		}
	}

	palette := make([]color.Color, 0, len(colors))
	for c := range colors {
		palette = append(palette, c)
	}
	return palette
}

// Magic
func generateWheelGIF(w io.Writer, options []string, target int, frames int, spins int, duration int) error {
	const width, height = 512, 512
	const radius = 200

	cx, cy := float64(width)/2, float64(height)/2

	wheelDC := gg.NewContext(width, height)
	wheelDC.SetRGBA(0, 0, 0, 0)
	wheelDC.Clear()
	drawWheel(wheelDC, options, cx, cy, radius)
	wheel := wheelDC.Image()

	arrowDC := gg.NewContext(width, height)
	wheelDC.SetRGBA(0, 0, 0, 0)
	arrowDC.Clear()
	drawArrow(arrowDC, cx, cy, radius)
	arrow := arrowDC.Image()

	// Sample the pieces into a single image to compute the palette for the GIF
	sampleDC := gg.NewContext(width, height)
	sampleDC.SetRGBA(0, 0, 0, 0)
	sampleDC.Clear()
	sampleDC.DrawImage(wheel, 0, 0)
	sampleDC.DrawImage(arrow, 0, 0)
	sample := sampleDC.Image()
	palette := createPalette(sample, 256)

	required := (float64(spins) * 2 * math.Pi) + calculateRotationOffset(options, target)

	var images []*image.Paletted
	var delays []int

	delay := clamp(100*duration/frames, 2, 50)

	for frame := 0; frame < frames; frame++ {
		progress := float64(frame) / float64(frames-1)
		eased := 1 - math.Pow(1-progress, 3)
		current := required * eased

		frameDC := gg.NewContext(width, height)
		wheelDC.SetRGBA(0, 0, 0, 0)
		frameDC.Clear()

		frameDC.Push()
		frameDC.Translate(cx, cy)
		frameDC.Rotate(current)
		frameDC.Translate(-cx, -cy)
		frameDC.DrawImage(wheel, 0, 0)
		frameDC.Pop()

		frameDC.DrawImage(arrow, 0, 0)

		frame := frameDC.Image()
		bounds := frame.Bounds()

		paletted := image.NewPaletted(bounds, palette)
		draw.FloydSteinberg.Draw(paletted, bounds, frame, bounds.Min)

		images = append(images, paletted)
		delays = append(delays, delay)
	}

	return gif.EncodeAll(w, &gif.GIF{
		Image: images,
		Delay: delays,
	})
}

func handler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	optionsParam := query.Get("options")
	targetParam := query.Get("target")
	if optionsParam == "" {
		http.Error(w, "Missing 'options' query parameter", http.StatusBadRequest)
		return
	}
	if targetParam == "" {
		http.Error(w, "Missing 'target' query parameter", http.StatusBadRequest)
		return
	}

	options := strings.Split(optionsParam, ",")
	if len(options) < 2 {
		http.Error(w, "Provide at least two options", http.StatusBadRequest)
		return
	}

	target, err := strconv.Atoi(targetParam)
	if err != nil || target < 0 || target >= len(options) {
		http.Error(w, "Invalid 'target' index", http.StatusBadRequest)
		return
	}

	frames := 60
	if f := query.Get("frames"); f != "" {
		frames, err = strconv.Atoi(f)
		if err != nil {
			http.Error(w, "Invalid 'frames' value (must be a number)", http.StatusBadRequest)
			return
		}
		frames = clamp(frames, 30, 120)
	}

	spins := 3
	if s := query.Get("spins"); s != "" {
		spins, err = strconv.Atoi(s)
		if err != nil {
			http.Error(w, "Invalid 'spins' value (must be a number)", http.StatusBadRequest)
			return
		}
		spins = clamp(spins, 1, 10)
	}

	duration := 3
	if d := query.Get("duration"); d != "" {
		duration, err = strconv.Atoi(d)
		if err != nil {
			http.Error(w, "Invalid 'duration' value (must be a number)", http.StatusBadRequest)
			return
		}
		duration = clamp(duration, 1, 10)
	}

	var buf bytes.Buffer
	err = generateWheelGIF(&buf, options, target, frames, spins, duration)
	if err != nil {
		log.Printf("Error generating wheel GIF: %v", err)
		http.Error(w, "Failed to generate wheel", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/gif")
	w.Write(buf.Bytes())
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Failed to load environment variables")
	}

	port := os.Getenv("PORT")
	if port == "" {
		fmt.Println("No port has been configured in environment variables")
		return
	}

	addr := fmt.Sprintf(":%s", port)

	http.HandleFunc("/", handler)

	fmt.Printf("Server running on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(addr, nil))
}
