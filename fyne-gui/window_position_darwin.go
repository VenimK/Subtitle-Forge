//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa

#import <Cocoa/Cocoa.h>

// getMainWindowPosition returns the x, y position of the main window
void getMainWindowPosition(double *x, double *y) {
    NSApplication *app = [NSApplication sharedApplication];
    NSWindow *window = [app mainWindow];
    if (window == nil && [[app windows] count] > 0) {
        window = [[app windows] objectAtIndex:0];
    }
    if (window != nil) {
        NSRect frame = [window frame];
        *x = frame.origin.x;
        *y = frame.origin.y;
    } else {
        *x = -1;
        *y = -1;
    }
}

// setMainWindowPosition moves the main window to the given x, y position
void setMainWindowPosition(double x, double y) {
    NSApplication *app = [NSApplication sharedApplication];
    NSWindow *window = [app mainWindow];
    if (window == nil && [[app windows] count] > 0) {
        window = [[app windows] objectAtIndex:0];
    }
    if (window != nil) {
        NSPoint point = NSMakePoint(x, y);
        [window setFrameOrigin:point];
    }
}
*/
import "C"

// GetWindowPosition returns the current main window position (x, y).
// Returns (-1, -1) if the window is not available.
func GetWindowPosition() (float64, float64) {
	var x, y C.double
	C.getMainWindowPosition(&x, &y)
	return float64(x), float64(y)
}

// SetWindowPosition moves the main window to the specified position.
func SetWindowPosition(x, y float64) {
	C.setMainWindowPosition(C.double(x), C.double(y))
}
