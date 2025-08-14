package assets

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed icon.png
var iconData []byte

// AppIcon returns the application icon
func AppIcon() fyne.Resource {
	return fyne.NewStaticResource("icon", iconData)
}
