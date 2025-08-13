package main

import (
	"fmt"
	"image/color"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// ColorPicker is a custom widget for selecting colors
type ColorPicker struct {
	widget.BaseWidget
	container    *fyne.Container
	colorPreview *canvas.Rectangle
	redSlider    *widget.Slider
	greenSlider  *widget.Slider
	blueSlider   *widget.Slider
	alphaSlider  *widget.Slider
	redLabel     *widget.Label
	greenLabel   *widget.Label
	blueLabel    *widget.Label
	alphaLabel   *widget.Label
	hexEntry     *widget.Entry
	currentColor color.NRGBA
	onChange     func(color.NRGBA)
}

// NewColorPicker creates a new color picker widget
func NewColorPicker(initialColor color.NRGBA, onChange func(color.NRGBA)) *ColorPicker {
	picker := &ColorPicker{
		currentColor: initialColor,
		onChange:     onChange,
	}
	picker.ExtendBaseWidget(picker)
	picker.createUI()
	return picker
}

// CreateRenderer is a private method to Fyne which defines how this widget
// is to be rendered.
func (c *ColorPicker) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(c.container)
}

// SetColor updates the color picker to show the specified color
func (c *ColorPicker) SetColor(col color.NRGBA) {
	c.currentColor = col
	c.updateUI()
}

// Color returns the currently selected color
func (c *ColorPicker) Color() color.NRGBA {
	return c.currentColor
}

func (c *ColorPicker) createUI() {
	// Create color preview rectangle
	c.colorPreview = canvas.NewRectangle(c.currentColor)
	c.colorPreview.SetMinSize(fyne.NewSize(100, 50))

	// Create sliders for RGB and Alpha values
	c.redSlider = widget.NewSlider(0, 255)
	c.redSlider.Step = 1
	c.redSlider.Value = float64(c.currentColor.R)
	c.redSlider.OnChanged = func(value float64) {
		c.currentColor.R = uint8(value)
		c.updateColorFromSliders()
	}

	c.greenSlider = widget.NewSlider(0, 255)
	c.greenSlider.Step = 1
	c.greenSlider.Value = float64(c.currentColor.G)
	c.greenSlider.OnChanged = func(value float64) {
		c.currentColor.G = uint8(value)
		c.updateColorFromSliders()
	}

	c.blueSlider = widget.NewSlider(0, 255)
	c.blueSlider.Step = 1
	c.blueSlider.Value = float64(c.currentColor.B)
	c.blueSlider.OnChanged = func(value float64) {
		c.currentColor.B = uint8(value)
		c.updateColorFromSliders()
	}

	c.alphaSlider = widget.NewSlider(0, 255)
	c.alphaSlider.Step = 1
	c.alphaSlider.Value = float64(c.currentColor.A)
	c.alphaSlider.OnChanged = func(value float64) {
		c.currentColor.A = uint8(value)
		c.updateColorFromSliders()
	}

	// Create labels to display RGB and Alpha values
	c.redLabel = widget.NewLabel(fmt.Sprintf("R: %d", c.currentColor.R))
	c.greenLabel = widget.NewLabel(fmt.Sprintf("G: %d", c.currentColor.G))
	c.blueLabel = widget.NewLabel(fmt.Sprintf("B: %d", c.currentColor.B))
	c.alphaLabel = widget.NewLabel(fmt.Sprintf("A: %d", c.currentColor.A))

	// Create hex entry for direct hex color input
	c.hexEntry = widget.NewEntry()
	c.hexEntry.SetText(fmt.Sprintf("#%02X%02X%02X%02X", c.currentColor.R, c.currentColor.G, c.currentColor.B, c.currentColor.A))
	c.hexEntry.OnSubmitted = func(s string) {
		if col, ok := c.parseHexColor(s); ok {
			c.currentColor = col
			c.updateUI()
		}
	}

	// Create preset color buttons
	presetColors := []color.NRGBA{
		{R: 255, G: 0, B: 0, A: 255},     // Red
		{R: 0, G: 255, B: 0, A: 255},     // Green
		{R: 0, G: 0, B: 255, A: 255},     // Blue
		{R: 255, G: 255, B: 0, A: 255},   // Yellow
		{R: 255, G: 0, B: 255, A: 255},   // Magenta
		{R: 0, G: 255, B: 255, A: 255},   // Cyan
		{R: 255, G: 255, B: 255, A: 255}, // White
		{R: 0, G: 0, B: 0, A: 255},       // Black
	}

	presetButtons := container.NewGridWithColumns(4)
	for _, presetColor := range presetColors {
		col := presetColor // Create a local copy for the closure
		btn := widget.NewButton("", func() {
			c.currentColor = col
			c.updateUI()
		})
		rect := canvas.NewRectangle(col)
		rect.SetMinSize(fyne.NewSize(30, 30))
		btnContainer := container.NewStack(btn, rect)
		presetButtons.Add(btnContainer)
	}

	// Assemble the UI
	sliderGrid := container.New(layout.NewFormLayout(),
		c.redLabel, c.redSlider,
		c.greenLabel, c.greenSlider,
		c.blueLabel, c.blueSlider,
		c.alphaLabel, c.alphaSlider,
		widget.NewLabel("Hex:"), c.hexEntry,
	)

	c.container = container.NewVBox(
		container.NewCenter(c.colorPreview),
		widget.NewLabel("Adjust Color:"),
		sliderGrid,
		widget.NewLabel("Presets:"),
		presetButtons,
	)
}

func (c *ColorPicker) updateUI() {
	// Update sliders
	c.redSlider.SetValue(float64(c.currentColor.R))
	c.greenSlider.SetValue(float64(c.currentColor.G))
	c.blueSlider.SetValue(float64(c.currentColor.B))
	c.alphaSlider.SetValue(float64(c.currentColor.A))

	// Update labels
	c.redLabel.SetText(fmt.Sprintf("R: %d", c.currentColor.R))
	c.greenLabel.SetText(fmt.Sprintf("G: %d", c.currentColor.G))
	c.blueLabel.SetText(fmt.Sprintf("B: %d", c.currentColor.B))
	c.alphaLabel.SetText(fmt.Sprintf("A: %d", c.currentColor.A))

	// Update hex entry
	c.hexEntry.SetText(fmt.Sprintf("#%02X%02X%02X%02X", c.currentColor.R, c.currentColor.G, c.currentColor.B, c.currentColor.A))

	// Update color preview
	c.colorPreview.FillColor = c.currentColor
	c.colorPreview.Refresh()

	// Call the onChange callback if provided
	if c.onChange != nil {
		c.onChange(c.currentColor)
	}
}

func (c *ColorPicker) updateColorFromSliders() {
	// Update the color preview
	c.colorPreview.FillColor = c.currentColor
	c.colorPreview.Refresh()

	// Update the labels
	c.redLabel.SetText(fmt.Sprintf("R: %d", c.currentColor.R))
	c.greenLabel.SetText(fmt.Sprintf("G: %d", c.currentColor.G))
	c.blueLabel.SetText(fmt.Sprintf("B: %d", c.currentColor.B))
	c.alphaLabel.SetText(fmt.Sprintf("A: %d", c.currentColor.A))

	// Update the hex entry
	c.hexEntry.SetText(fmt.Sprintf("#%02X%02X%02X%02X", c.currentColor.R, c.currentColor.G, c.currentColor.B, c.currentColor.A))

	// Call the onChange callback if provided
	if c.onChange != nil {
		c.onChange(c.currentColor)
	}
}

func (c *ColorPicker) parseHexColor(s string) (color.NRGBA, bool) {
	var col color.NRGBA

	// Remove the # prefix if present
	if len(s) > 0 && s[0] == '#' {
		s = s[1:]
	}

	// Parse the hex color
	switch len(s) {
	case 6: // RGB format
		r, err1 := strconv.ParseUint(s[0:2], 16, 8)
		g, err2 := strconv.ParseUint(s[2:4], 16, 8)
		b, err3 := strconv.ParseUint(s[4:6], 16, 8)
		if err1 == nil && err2 == nil && err3 == nil {
			col = color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
			return col, true
		}
	case 8: // RGBA format
		r, err1 := strconv.ParseUint(s[0:2], 16, 8)
		g, err2 := strconv.ParseUint(s[2:4], 16, 8)
		b, err3 := strconv.ParseUint(s[4:6], 16, 8)
		a, err4 := strconv.ParseUint(s[6:8], 16, 8)
		if err1 == nil && err2 == nil && err3 == nil && err4 == nil {
			col = color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(a)}
			return col, true
		}
	}

	return col, false
}
