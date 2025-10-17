#!/bin/bash

echo "🚀 Launching Subtitle Forge v1.9.2 test build..."
echo ""

# Check if test build exists
if [ ! -f "fyne-gui/test-build" ]; then
    echo "❌ Test build not found! Run ./test-build.sh first"
    exit 1
fi

echo "🎯 Testing the new v1.9.2 track sorting feature:"
echo "   1. Load multiple MKV files using 'Select Multiple Files (Batch)'"
echo "   2. Try the new 'Sort by:' dropdown with options:"
echo "      - Default Order"
echo "      - By Filename" 
echo "      - By Language"
echo "      - By Codec"
echo "      - By Track Number"
echo "   3. Combine filtering and sorting for powerful track management"
echo ""
echo "🔍 Other features to verify:"
echo "   - Track filtering with filename display"
echo "   - Subtitle extraction and conversion"
echo "   - Insert subtitles functionality"
echo ""

# Launch the test build
cd fyne-gui
./test-build

echo ""
echo "✅ Test session completed!"
