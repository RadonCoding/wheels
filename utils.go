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

func interpolate(c1, c2 color.RGBA, factor float64) color.RGBA {
	r := uint8(float64(c1.R)*(1-factor) + float64(c2.R)*factor)
	g := uint8(float64(c1.G)*(1-factor) + float64(c2.G)*factor)
	b := uint8(float64(c1.B)*(1-factor) + float64(c2.B)*factor)
	a := uint8(float64(c1.A)*(1-factor) + float64(c2.A)*factor)
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
