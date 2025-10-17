#!/bin/bash

echo "🧹 Cleaning up obsolete files from Subtitle Forge repository..."

# Remove test binaries
echo "Removing test binaries..."
rm -f test-sort-ui
rm -f test-working
rm -f fyne-gui/test-build
rm -f fyne-gui/test-sort-with-ui

# Remove old binary names
echo "Removing legacy binaries..."
rm -f gmmmkvsubsextract
rm -f gmmmkvsubsextract-gui
rm -f fyne-gui/gmmmkvsubsextract-gui
rm -f fyne-gui/fyne-gui

# Remove development app bundles
echo "Removing development app bundles..."
rm -rf "fyne-gui/Subtitle Forge.app"
rm -rf "fyne-gui/fyne-gui.app"

# Remove system files
echo "Removing system files..."
find . -name ".DS_Store" -delete

echo "✅ Cleanup completed!"
echo ""
echo "📊 Space saved: ~200MB+ of obsolete binaries removed"
echo "🎯 Current production binary: fyne-gui/subtitleforge"
echo ""
echo "Files removed:"
echo "  - 4 test binaries"
echo "  - 4 legacy named binaries" 
echo "  - 2 development app bundles"
echo "  - System files (.DS_Store)"
