#!/bin/bash

echo "🔧 Running test build for Subtitle Forge v1.9.2..."
echo ""

# Change to fyne-gui directory
cd fyne-gui

# Check if we have all necessary files
echo "📋 Checking source files..."
if [ ! -f "main.go" ]; then
    echo "❌ main.go not found!"
    exit 1
fi

if [ ! -f "go.mod" ]; then
    echo "❌ go.mod not found!"
    exit 1
fi

echo "✅ Source files present"
echo ""

# Clean any existing test build
echo "🧹 Cleaning previous test builds..."
rm -f test-build

# Run the test build
echo "🔨 Building test binary..."
go build -o test-build

# Check if build was successful
if [ $? -eq 0 ]; then
    echo "✅ Test build successful!"
    
    # Check binary size
    if [ -f "test-build" ]; then
        size=$(du -h test-build | cut -f1)
        echo "📊 Binary size: $size"
        
        # Make it executable
        chmod +x test-build
        
        echo ""
        echo "🚀 Test build ready!"
        echo "   Location: fyne-gui/test-build"
        echo "   To run: cd fyne-gui && ./test-build"
        echo ""
        echo "🎯 Features to test:"
        echo "   - Load MKV files (single and batch)"
        echo "   - Filter tracks by language/codec/filename"
        echo "   - Sort tracks (new v1.9.2 feature!)"
        echo "   - Extract subtitles"
        echo "   - Convert subtitle formats"
        echo "   - Insert subtitles into MKV"
        
    else
        echo "❌ Binary not created despite successful build"
        exit 1
    fi
else
    echo "❌ Build failed!"
    exit 1
fi
