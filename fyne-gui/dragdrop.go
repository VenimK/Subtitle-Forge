package main

import (
	"image/color"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// FileDropArea is a custom widget that supports drag and drop for files
type FileDropArea struct {
	widget.BaseWidget
	rect       *canvas.Rectangle
	descLabel  *widget.Label
	fileLabel  *widget.Label
	content    *fyne.Container
	extensions []string
	onDropped  func(string)
}

// NewFileDropArea creates a new file drop area widget
func NewFileDropArea(description string, extensions []string, onDropped func(string)) *FileDropArea {
	dropArea := &FileDropArea{
		extensions: extensions,
		onDropped:  onDropped,
	}

	dropArea.ExtendBaseWidget(dropArea)

	// Create visual elements
	dropArea.rect = canvas.NewRectangle(color.NRGBA{R: 200, G: 200, B: 200, A: 100})
	dropArea.descLabel = widget.NewLabel(description)
	dropArea.descLabel.Alignment = fyne.TextAlignCenter
	dropArea.fileLabel = widget.NewLabel("Drop file here")
	dropArea.fileLabel.Alignment = fyne.TextAlignCenter

	// Create layout
	dropArea.content = container.NewStack(
		dropArea.rect,
		container.NewVBox(
			dropArea.descLabel,
			dropArea.fileLabel,
		),
	)

	return dropArea
}

// CreateRenderer implements fyne.Widget
func (d *FileDropArea) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(d.content)
}

// SetMinSize sets the minimum size of the widget
func (d *FileDropArea) SetMinSize(size fyne.Size) {
	d.content.Resize(size)
}

// MinSize returns the minimum size of the widget
func (d *FileDropArea) MinSize() fyne.Size {
	return d.content.MinSize()
}

// Resize handles resizing the widget
func (d *FileDropArea) Resize(size fyne.Size) {
	d.BaseWidget.Resize(size)
	d.content.Resize(size)
}

// Move handles moving the widget
func (d *FileDropArea) Move(pos fyne.Position) {
	d.BaseWidget.Move(pos)
	d.content.Move(pos)
}

// DragEnter implements desktop.Hoverable
func (d *FileDropArea) DragEnter() {
	d.rect.FillColor = color.NRGBA{R: 100, G: 200, B: 100, A: 100}
	d.rect.Refresh()
}

// DragLeave implements desktop.Hoverable
func (d *FileDropArea) DragLeave() {
	d.rect.FillColor = color.NRGBA{R: 200, G: 200, B: 200, A: 100}
	d.rect.Refresh()
}

// AcceptDroppedFiles implements fyne.DroppableFiles
func (d *FileDropArea) AcceptDroppedFiles() bool {
	return true
}

// DropFile implements fyne.DroppableFiles
func (d *FileDropArea) DropFile(file fyne.URIReadCloser) {
	if file == nil {
		return
	}

	path := file.URI().Path()
	file.Close() // Close the file as we only need the path

	// Check if it has a valid extension
	ext := strings.ToLower(filepath.Ext(path))
	valid := false
	for _, validExt := range d.extensions {
		if ext == validExt {
			valid = true
			break
		}
	}

	if valid {
		d.fileLabel.SetText(filepath.Base(path))
		if d.onDropped != nil {
			d.onDropped(path)
		}
	} else {
		d.fileLabel.SetText("Invalid file type")
	}
}

// MouseIn implements desktop.Hoverable
func (d *FileDropArea) MouseIn(*desktop.MouseEvent) {
	// Optional hover effect
}

// MouseOut implements desktop.Hoverable
func (d *FileDropArea) MouseOut() {
	// Optional hover effect
}

// MouseMoved implements desktop.Hoverable
func (d *FileDropArea) MouseMoved(*desktop.MouseEvent) {
	// Optional hover effect
}

// Ensure our widget implements the necessary interfaces
var _ fyne.Widget = (*FileDropArea)(nil)
var _ fyne.Draggable = (*FileDropArea)(nil)
var _ desktop.Hoverable = (*FileDropArea)(nil)

// Dragged implements fyne.Draggable
func (d *FileDropArea) Dragged(e *fyne.DragEvent) {
	// Not used for file drop areas
}

// DragEnd implements fyne.Draggable
func (d *FileDropArea) DragEnd() {
	// Not used for file drop areas
}
