package main

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

// TestRefreshTrackListKeepsRowsVisible covers the track list being refilled with
// fewer rows than it had while the user had scrolled down. Refreshing only the
// row container leaves the scroll at its old offset, which paints the surviving
// rows above the viewport and the list looks empty.
func TestRefreshTrackListKeepsRowsVisible(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()
	win := test.NewWindow(nil)
	defer win.Close()

	list := container.NewVBox()
	scroll := container.NewScroll(list)
	scroll.SetMinSize(fyne.NewSize(400, 250))
	win.SetContent(scroll)
	win.Resize(fyne.NewSize(600, 400))
	win.Show()

	for i := 0; i < 40; i++ {
		list.Add(container.NewHBox(widget.NewLabel(fmt.Sprintf("Track %d", i))))
	}
	refreshTrackList(list, scroll)
	if scroll.Size().Height >= scroll.Content.MinSize().Height {
		t.Fatalf("test setup: 40 rows should overflow the scroll (scroll %v, content min %v)",
			scroll.Size(), scroll.Content.MinSize())
	}

	// The state a user leaves the scroll in after scrolling to a later track.
	scroll.Offset = fyne.NewPos(0, 180)

	list.Objects = nil
	list.Add(container.NewHBox(widget.NewLabel("Track 28")))
	refreshTrackList(list, scroll)

	if scroll.Offset.Y != 0 {
		t.Errorf("scroll offset after refill = %v, want (0, 0) so the first row is on screen", scroll.Offset)
	}
	if scroll.Content.MinSize().Height > scroll.Size().Height {
		t.Errorf("content min %v still exceeds the scroll %v after the refill",
			scroll.Content.MinSize(), scroll.Size())
	}
	row := list.Objects[0]
	top := row.Position().Y - scroll.Offset.Y
	if top < 0 || top >= scroll.Size().Height {
		t.Errorf("row %v sits at visible y=%v, outside the %v tall viewport", row.Size(), top, scroll.Size())
	}
}
