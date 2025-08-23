#!/bin/bash

echo "Setting up tessdata_best for OCR..."
TESSDATA_DIR=~/tessdata_best

# Create tessdata_best directory in home if it doesn't exist
if [ ! -d "$TESSDATA_DIR" ]; then
    mkdir -p "$TESSDATA_DIR"
    echo "Created $TESSDATA_DIR directory"
else
    echo "$TESSDATA_DIR directory already exists"
fi

# Check if we need to download language models
if [ "$(ls -A $TESSDATA_DIR)" ]; then
    echo "Language data files found in $TESSDATA_DIR"
else
    echo "No language data found. Downloading commonly used language models..."
    
    # Download common language models from tessdata_best
    cd "$TESSDATA_DIR"
    
    # Common language models
    langs=("eng" "fra" "deu" "spa" "ita" "jpn" "kor" "chi_sim" "chi_tra" "rus")
    
    for lang in "${langs[@]}"; do
        echo "Downloading $lang.traineddata..."
        curl -L "https://github.com/tesseract-ocr/tessdata_best/raw/main/$lang.traineddata" -o "$lang.traineddata"
    done
    
    echo "Language models downloaded to $TESSDATA_DIR"
fi

# Set environment variable for current session
export TESSDATA_PREFIX="$TESSDATA_DIR"
echo "Set TESSDATA_PREFIX=$TESSDATA_DIR for this session"
echo "For permanent use, add this to your .bashrc or .zshrc:"
echo "  export TESSDATA_PREFIX=$TESSDATA_DIR"
echo ""
echo "Note: The application will automatically set TESSDATA_PREFIX when running."

exit 0
