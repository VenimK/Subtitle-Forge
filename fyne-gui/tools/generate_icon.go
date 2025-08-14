package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"math"
	"os"
)

func main() {
	// Create a 256x256 RGBA image
	img := image.NewRGBA(image.Rect(0, 0, 256, 256))

	// Fill background with dark blue
	bgColor := color.RGBA{30, 50, 92, 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)

	// Draw a lighter circle in the center
	circleColor := color.RGBA{60, 100, 180, 255}
	drawCircle(img, 128, 128, 100, circleColor)

	// Draw subtitle lines
	lineColor := color.RGBA{240, 240, 240, 255}
	drawLine(img, 64, 140, 192, 140, 6, lineColor)
	drawLine(img, 64, 170, 160, 170, 6, lineColor)

	// Draw a checkmark
	checkColor := color.RGBA{50, 200, 50, 255}
	drawLine(img, 100, 110, 120, 130, 8, checkColor)
	drawLine(img, 120, 130, 160, 90, 8, checkColor)

	// Save the image to a file
	f, err := os.Create("assets/icon.png")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		log.Fatal(err)
	}

	log.Println("Icon generated successfully at assets/icon.png")
}

// drawCircle draws a filled circle
func drawCircle(img *image.RGBA, x, y, r int, c color.RGBA) {
	for i := -r; i <= r; i++ {
		for j := -r; j <= r; j++ {
			if i*i+j*j <= r*r {
				img.Set(x+i, y+j, c)
			}
		}
	}
}

// drawLine draws a line with specified thickness
func drawLine(img *image.RGBA, x1, y1, x2, y2, thickness int, c color.RGBA) {
	dx := float64(x2 - x1)
	dy := float64(y2 - y1)
	dist := math.Sqrt(dx*dx + dy*dy)
	
	if dist == 0 {
		return
	}
	
	dx /= dist
	dy /= dist
	
	// Draw the line with thickness
	for t := 0.0; t < dist; t += 0.5 {
		x := int(float64(x1) + dx*t)
		y := int(float64(y1) + dy*t)
		
		for i := -thickness/2; i <= thickness/2; i++ {
			for j := -thickness/2; j <= thickness/2; j++ {
				if i*i+j*j <= (thickness*thickness)/4 {
					img.Set(x+i, y+j, c)
				}
			}
		}
	}
}
