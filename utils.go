package main

import (
	"image/color"
	"math"
)

func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func interpolate(c1, c2 color.Color, factor float64) color.RGBA {
	r1, g1, b1, a1 := c1.RGBA()
	r2, g2, b2, a2 := c2.RGBA()
	convert := func(x uint32) float64 {
		return float64(x) / 257.0
	}
	r := uint8(convert(r1)*(1-factor) + convert(r2)*factor)
	g := uint8(convert(g1)*(1-factor) + convert(g2)*factor)
	b := uint8(convert(b1)*(1-factor) + convert(b2)*factor)
	a := uint8(convert(a1)*(1-factor) + convert(a2)*factor)
	return color.RGBA{r, g, b, a}
}

// Uses the same logic as the wheel drawing to calculate how much to rotate to reach a specific option
func clockWiseToTarget(options []string, target int) float64 {
	angleStep := 2 * math.Pi / float64(len(options))
	startAngle := angleStep * float64(target)
	endAngle := startAngle + angleStep
	midAngle := (startAngle + endAngle) / 2
	return math.Mod((3*math.Pi/2)-midAngle+2*math.Pi, 2*math.Pi)
}
