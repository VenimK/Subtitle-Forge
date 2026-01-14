#!/bin/bash
# ═══════════════════════════════════════════════════════════════════
# Batch SRT Translation Script for LibreTranslate
# ═══════════════════════════════════════════════════════════════════

set -e

# Configuration (edit these or pass as arguments)
API_URL="${LIBRETRANSLATE_URL:-http://localhost:5000}"
API_KEY="${LIBRETRANSLATE_API_KEY:-}"
SOURCE_LANG="${SOURCE_LANG:-en}"
TARGET_LANG="${TARGET_LANG:-nl}"
PARALLEL_JOBS="${PARALLEL_JOBS:-4}"
OUTPUT_DIR="${OUTPUT_DIR:-./translated}"
SAME_DIR=false

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

usage() {
    cat <<EOF
Usage: $(basename "$0") [OPTIONS] <input_files_or_directory>

Batch translate SRT subtitle files using LibreTranslate API.

Options:
  -u, --url URL         LibreTranslate API URL (default: http://localhost:5000)
  -k, --api-key KEY     API key for authentication
  -s, --source LANG     Source language code (default: en)
  -t, --target LANG     Target language code (default: nl)
  -o, --output DIR      Output directory (default: ./translated)
  --same-dir            Save output next to input file (ignores -o)
  -j, --jobs N          Number of parallel jobs (default: 4)
  -h, --help            Show this help message

Examples:
  $(basename "$0") -k YOUR_API_KEY -s en -t es *.srt
  $(basename "$0") -k YOUR_API_KEY -t de ./subtitles/
  $(basename "$0") --url http://192.168.1.49:5000 -k KEY movie.srt

Environment variables:
  LIBRETRANSLATE_URL      API URL
  LIBRETRANSLATE_API_KEY  API key
  SOURCE_LANG             Source language
  TARGET_LANG             Target language
  PARALLEL_JOBS           Number of parallel jobs
  OUTPUT_DIR              Output directory

EOF
    exit 0
}

# Parse arguments
POSITIONAL=()
while [[ $# -gt 0 ]]; do
    case $1 in
        -u|--url)
            API_URL="$2"
            shift 2
            ;;
        -k|--api-key)
            API_KEY="$2"
            shift 2
            ;;
        -s|--source)
            SOURCE_LANG="$2"
            shift 2
            ;;
        -t|--target)
            TARGET_LANG="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        --same-dir)
            SAME_DIR=true
            shift
            ;;
        -j|--jobs)
            PARALLEL_JOBS="$2"
            shift 2
            ;;
        -h|--help)
            usage
            ;;
        -*)
            echo -e "${RED}Unknown option: $1${NC}"
            usage
            ;;
        *)
            POSITIONAL+=("$1")
            shift
            ;;
    esac
done

set -- "${POSITIONAL[@]}"

# Validate inputs
if [ ${#POSITIONAL[@]} -eq 0 ]; then
    echo -e "${RED}Error: No input files specified${NC}"
    usage
fi

# Collect all SRT files
FILES=()
for input in "${POSITIONAL[@]}"; do
    if [ -d "$input" ]; then
        while IFS= read -r -d '' file; do
            FILES+=("$file")
        done < <(find "$input" -name "*.srt" -type f -print0)
    elif [ -f "$input" ]; then
        FILES+=("$input")
    else
        echo -e "${YELLOW}Warning: '$input' not found, skipping${NC}"
    fi
done

if [ ${#FILES[@]} -eq 0 ]; then
    echo -e "${RED}Error: No SRT files found${NC}"
    exit 1
fi

# Create output directory
mkdir -p "$OUTPUT_DIR"

echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  LibreTranslate Batch SRT Translation${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "  API URL:     ${GREEN}$API_URL${NC}"
echo -e "  Source:      ${GREEN}$SOURCE_LANG${NC}"
echo -e "  Target:      ${GREEN}$TARGET_LANG${NC}"
echo -e "  Files:       ${GREEN}${#FILES[@]}${NC}"
echo -e "  Parallel:    ${GREEN}$PARALLEL_JOBS${NC}"
echo -e "  Output:      ${GREEN}$OUTPUT_DIR${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo ""

# Check API connectivity
echo -n "Checking API connectivity... "
if curl -s --max-time 5 "$API_URL/languages" > /dev/null 2>&1; then
    echo -e "${GREEN}OK${NC}"
else
    echo -e "${RED}FAILED${NC}"
    echo -e "${RED}Cannot connect to $API_URL${NC}"
    exit 1
fi

# Translation function
translate_file() {
    local file="$1"
    local api_url="$2"
    local api_key="$3"
    local source_lang="$4"
    local target_lang="$5"
    local output_dir="$6"
    local same_dir="$7"
    
    local filename=$(basename "$file")
    local name="${filename%.*}"
    
    # Determine output location
    local actual_output_dir="$output_dir"
    if [ "$same_dir" = "true" ]; then
        actual_output_dir="$(dirname "$file")"
    fi
    
    local output_file="${actual_output_dir}/${name}.${target_lang}.srt"
    local temp_response="${actual_output_dir}/.response_$$.json"
    
    # Ensure output directory exists
    mkdir -p "$actual_output_dir"
    
    echo -e "${BLUE}[INFO]${NC} Translating: ${filename}"
    echo "PROGRESS:0" >> "$OUTPUT_DIR/.progress"
    
    # Build curl command
    local curl_args=(
        -s
        -X POST
        "$api_url/translate_file"
        -F "file=@$file"
        -F "source=$source_lang"
        -F "target=$target_lang"
    )
    
    if [ -n "$api_key" ]; then
        # Try both form field and header for compatibility
        curl_args+=(-F "api_key=$api_key")
        curl_args+=(-H "Authorization: Bearer $api_key")
    fi
    
    # Execute translation request
    local http_code
    http_code=$(curl "${curl_args[@]}" -o "$temp_response" -w "%{http_code}")
    
    if [ "$http_code" = "200" ] && [ -s "$temp_response" ]; then
        # Check if response contains a download URL (JSON) or direct content
        if grep -q "translatedFileUrl" "$temp_response" 2>/dev/null; then
            # Extract download URL from JSON and download the actual file
            local download_url
            download_url=$(grep -o '"translatedFileUrl":"[^"]*"' "$temp_response" | cut -d'"' -f4)
            
            if [ -n "$download_url" ]; then
                # If API URL uses HTTPS but download URL uses HTTP, convert it
                if [[ "$api_url" == https://* ]] && [[ "$download_url" == http://* ]]; then
                    download_url="${download_url/http:/https:}"
                fi
                
                # Download the actual translated file
                local dl_code
                dl_code=$(curl -s -L -o "$output_file" -w "%{http_code}" "$download_url")
                rm -f "$temp_response"
                
                if [ "$dl_code" = "200" ] && [ -s "$output_file" ]; then
                    echo -e "${GREEN}✓${NC} Translated: ${filename} -> $(basename "$output_file")"
                    echo "PROGRESS:1" >> "$OUTPUT_DIR/.progress"
                    return 0
                else
                    echo -e "${RED}✗${NC} Failed to download translated file for ${filename}"
                    echo "PROGRESS:1" >> "$OUTPUT_DIR/.progress"
                    rm -f "$output_file"
                    return 1
                fi
            else
                echo -e "${RED}✗${NC} $filename (no download URL in response)"
                rm -f "$temp_response"
                return 1
            fi
        else
            # Response is the actual file content (direct response)
            mv "$temp_response" "$output_file"
            echo -e "${GREEN}✓${NC} Translated: ${filename} -> $(basename "$output_file") (direct response)"
            echo "PROGRESS:1" >> "$OUTPUT_DIR/.progress"
            return 0
        fi
    else
        echo -e "${RED}✗${NC} Translation failed for ${filename} (HTTP $http_code)"
        echo "PROGRESS:1" >> "$OUTPUT_DIR/.progress"
        rm -f "$temp_response"
        return 1
    fi
}

export -f translate_file
export API_URL API_KEY SOURCE_LANG TARGET_LANG OUTPUT_DIR SAME_DIR
export RED GREEN YELLOW BLUE NC

# Process files
echo ""
echo "Translating files..."
echo ""

SUCCESS=0
FAILED=0
TOTAL=${#FILES[@]}
TOTAL_STEPS=$((TOTAL * 2))  # two steps per file: start + finish

# Progress function
update_progress() {
    local steps=$1
    local total=$2
    local width=40
    local percent=$((steps * 100 / total))
    local filled=$((steps * width / total))
    local empty=$((width - filled))
    printf "\r[%3d%%] [" "$percent"
    printf "%*s" "$filled" | tr ' ' '='
    printf "%*s" "$empty" | tr ' ' '-'
    printf "] %d/%d" "$((steps / 2))" "$((total / 2))"
}

# Initialize progress
update_progress 0 "$TOTAL_STEPS"
# Clean up any old progress file
rm -f "$OUTPUT_DIR/.progress"

if command -v parallel &> /dev/null; then
    # Use GNU parallel if available
    parallel -j "$PARALLEL_JOBS" translate_file {} "$API_URL" "$API_KEY" "$SOURCE_LANG" "$TARGET_LANG" "$OUTPUT_DIR" "$SAME_DIR" ::: "${FILES[@]}" &
else
    # Fallback to xargs (NUL-delimited to handle special chars)
    printf '%s\0' "${FILES[@]}" | xargs -0 -P "$PARALLEL_JOBS" -I {} bash -c 'translate_file "$@"' _ {} "$API_URL" "$API_KEY" "$SOURCE_LANG" "$TARGET_LANG" "$OUTPUT_DIR" "$SAME_DIR" &
fi

# Monitor progress
BG_PID=$!
sleep 0.2  # brief pause to allow initial progress markers
while [ -d "/proc/$BG_PID" ] 2>/dev/null || kill -0 "$BG_PID" 2>/dev/null; do
    steps=$(grep -c "PROGRESS:" "$OUTPUT_DIR/.progress" 2>/dev/null || echo 0)
    update_progress "$steps" "$TOTAL_STEPS"
    sleep 0.3
done
wait "$BG_PID"

# Final update (ensure 100%)
steps=$(grep -c "PROGRESS:" "$OUTPUT_DIR/.progress" 2>/dev/null || echo 0)
if [ "$steps" -lt "$TOTAL_STEPS" ]; then
    steps=$TOTAL_STEPS
fi
update_progress "$steps" "$TOTAL_STEPS"
echo ""  # newline after progress bar

# Count results
SUCCESS=$(grep -c "PROGRESS:1" "$OUTPUT_DIR/.progress" 2>/dev/null || echo 0)
FAILED=$((TOTAL - SUCCESS))
rm -f "$OUTPUT_DIR/.progress"

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "  Translation complete!"
echo -e "  Output directory: ${GREEN}$OUTPUT_DIR${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
