#!/usr/bin/env bash
set -euo pipefail

PREFIX_DIR="${PREFIX_DIR:-$HOME/.whispercpp-coreml}"
REPO_DIR="${REPO_DIR:-$PREFIX_DIR/whisper.cpp}"
BUILD_DIR="${BUILD_DIR:-$REPO_DIR/build}"
MODELS_DIR="${MODELS_DIR:-$REPO_DIR/models}"
VENV_DIR="${VENV_DIR:-$PREFIX_DIR/venv}"

MODEL_NAME_DEFAULT="${MODEL_NAME_DEFAULT:-base.en}"
MODEL_FILE_DEFAULT="${MODEL_FILE_DEFAULT:-$MODELS_DIR/ggml-${MODEL_NAME_DEFAULT}.bin}"

usage() {
  cat <<'EOF'
Usage:
  ./Whisper-mac-coreml.sh install
  ./Whisper-mac-coreml.sh install-deps
  ./Whisper-mac-coreml.sh list-models
  ./Whisper-mac-coreml.sh download-model [model_name]
  ./Whisper-mac-coreml.sh generate-coreml [model_name]
  ./Whisper-mac-coreml.sh convert <input_media> <output_wav>
  ./Whisper-mac-coreml.sh transcribe <input_media_or_wav> [--model <ggml_model_path>] [--out <output_prefix>] [--lang <code>] [--srt] [--vtt] [--txt]

Notes:
- Installs whisper.cpp with Core ML support for Apple Silicon (ANE acceleration).
- Requires Xcode command-line tools and Python 3.11+ (Core ML dependencies).
- After generating a Core ML model, whisper.cpp will use it automatically for the encoder.

Environment variables:
  PREFIX_DIR         (default: ~/.whispercpp-coreml)
  MODEL_NAME_DEFAULT (default: base.en)
EOF
}

list_models() {
  cat <<'EOF'
Supported whisper.cpp GGML model names (common):

English-only (smaller / faster):
  tiny.en
  base.en
  small.en
  medium.en

Multilingual (auto-detect language):
  tiny
  base
  small
  medium
  large-v1
  large-v2
  large-v3
  large-v3-turbo

Tips:
- Use *.en models if you ONLY need English (faster / smaller).
- Use non-.en models for other languages or mixed language content.
- Download example:
    ./Whisper-mac-coreml.sh download-model medium
- CoreML encoder example (Apple Silicon speedup):
    ./Whisper-mac-coreml.sh generate-coreml medium
EOF
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "Missing required command: $1" >&2
    exit 1
  }
}

ensure_brew_deps() {
  need_cmd brew
  brew install -q git cmake ffmpeg pkg-config || true
}

ensure_python_deps() {
  need_cmd python3
  # Prefer using a python3.11 if available; otherwise use system python3
  local PYTHON_CMD=""
  if command -v python3.11 >/dev/null 2>&1; then
    PYTHON_CMD=python3.11
  elif command -v python3 >/dev/null 2>&1; then
    PYTHON_CMD=python3
  else
    echo "Python3 not found. Please install Python 3.11+." >&2
    exit 1
  fi

  echo "Using Python: $("$PYTHON_CMD" --version)"

  # Create isolated venv for CoreML dependencies (avoids conflicts with other venvs)
  if [ ! -d "$VENV_DIR" ]; then
    echo "Creating isolated venv at $VENV_DIR..."
    "$PYTHON_CMD" -m venv "$VENV_DIR"
  fi

  # Use venv Python for all installs
  local VENV_PY="$VENV_DIR/bin/python"
  echo "Installing CoreML dependencies into venv..."
  "$VENV_PY" -m pip install --upgrade pip setuptools wheel
  "$VENV_PY" -m pip install 'numpy<2' torch torchvision torchaudio
  "$VENV_PY" -m pip install ane_transformers openai-whisper coremltools
}

install_whispercpp() {
  if ! command -v brew >/dev/null 2>&1; then
    echo "Homebrew not found. Install it first:" >&2
    echo "  https://brew.sh" >&2
    exit 1
  fi

  ensure_brew_deps
  ensure_python_deps

  mkdir -p "$PREFIX_DIR"

  if [ ! -d "$REPO_DIR/.git" ]; then
    git clone https://github.com/ggml-org/whisper.cpp.git "$REPO_DIR"
  else
    git -C "$REPO_DIR" pull --ff-only
  fi

  # Build with Core ML support
  cmake -S "$REPO_DIR" -B "$BUILD_DIR" -DWHISPER_COREML=1
  cmake --build "$BUILD_DIR" -j --config Release

  echo "Install complete (with Core ML support)"
  echo "Binary: $BUILD_DIR/bin/whisper-cli"
  echo "Tip: run '$BUILD_DIR/bin/whisper-cli -h' for all options"
  echo ""
  echo "Next steps:"
  echo "  ./Whisper-mac-coreml.sh download-model"
  echo "  ./Whisper-mac-coreml.sh generate-coreml"
}

download_model() {
  local model_name="${1:-$MODEL_NAME_DEFAULT}"

  if [ ! -d "$REPO_DIR" ]; then
    echo "whisper.cpp not installed. Run: ./Whisper-mac-coreml.sh install" >&2
    exit 1
  fi

  (cd "$REPO_DIR" && sh ./models/download-ggml-model.sh "$model_name")
  echo "Model downloaded: $MODELS_DIR/ggml-${model_name}.bin"
}

generate_coreml_model() {
  local model_name="${1:-$MODEL_NAME_DEFAULT}"

  if [ ! -d "$REPO_DIR" ]; then
    echo "whisper.cpp not installed. Run: ./Whisper-mac-coreml.sh install" >&2
    exit 1
  fi

  if [ ! -f "$MODELS_DIR/ggml-${model_name}.bin" ]; then
    echo "GGML model not found. Run: ./Whisper-mac-coreml.sh download-model $model_name" >&2
    exit 1
  fi

  # Use the isolated venv Python (must have CoreML deps installed)
  local VENV_PY="$VENV_DIR/bin/python"
  if [ ! -x "$VENV_PY" ]; then
    echo "Venv not found. Run: ./Whisper-mac-coreml.sh install-deps" >&2
    exit 1
  fi

  echo "Generating Core ML model for $model_name using venv Python..."
  (cd "$REPO_DIR/models" && "$VENV_PY" ./convert-whisper-to-coreml.py --model "$model_name" --encoder-only True)
  
  # Compile the mlpackage to mlmodelc
  local mlpackage="$MODELS_DIR/models/coreml-encoder-${model_name}.mlpackage"
  local mlmodelc="$MODELS_DIR/ggml-${model_name}-encoder.mlmodelc"
  
  if [ -d "$mlpackage" ]; then
    echo "Compiling Core ML model..."
    xcrun coremlcompiler compile "$mlpackage" "$MODELS_DIR/"
    # Rename to expected path
    if [ -d "$MODELS_DIR/coreml-encoder-${model_name}.mlmodelc" ]; then
      rm -rf "$mlmodelc"
      mv "$MODELS_DIR/coreml-encoder-${model_name}.mlmodelc" "$mlmodelc"
    fi
  fi
  
  if [ -d "$mlmodelc" ]; then
    echo "Core ML model generated: $mlmodelc"
  else
    echo "Warning: Core ML model not found at expected path: $mlmodelc" >&2
  fi
}

convert_media() {
  local in="${1:-}"
  local out="${2:-}"

  if [ -z "$in" ] || [ -z "$out" ]; then
    echo "convert requires: <input_media> <output_wav>" >&2
    exit 1
  fi

  need_cmd ffmpeg
  ffmpeg -y -i "$in" -ar 16000 -ac 1 -c:a pcm_s16le "$out"
}

transcribe() {
  local input="${1:-}"
  shift || true

  if [ -z "$input" ]; then
    echo "transcribe requires: <input_media_or_wav>" >&2
    exit 1
  fi

  if [ ! -x "$BUILD_DIR/bin/whisper-cli" ]; then
    echo "whisper.cpp not built. Run: ./Whisper-mac-coreml.sh install" >&2
    exit 1
  fi

  local model_file="$MODEL_FILE_DEFAULT"
  local out_prefix=""
  local lang_code=""
  local want_srt=0
  local want_vtt=0
  local want_txt=0

  while [ $# -gt 0 ]; do
    case "$1" in
      --model)
        model_file="$2"
        shift 2
        ;;
      --out)
        out_prefix="$2"
        shift 2
        ;;
      --lang)
        lang_code="$2"
        shift 2
        ;;
      --srt)
        want_srt=1
        shift
        ;;
      --vtt)
        want_vtt=1
        shift
        ;;
      --txt)
        want_txt=1
        shift
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        echo "Unknown arg: $1" >&2
        exit 1
        ;;
    esac
  done

  if [ -z "$out_prefix" ]; then
    out_prefix="${input%.*}"
  fi

  if [ ! -f "$model_file" ]; then
    echo "Model file not found: $model_file" >&2
    echo "Run: ./Whisper-mac-coreml.sh download-model $MODEL_NAME_DEFAULT" >&2
    exit 1
  fi

  local wav_input="$input"
  local tmp_wav=""

  case "${input##*.}" in
    wav|WAV)
      ;;
    *)
      tmp_wav="$(mktemp -t whispercpp.XXXXXX).wav"
      convert_media "$input" "$tmp_wav"
      wav_input="$tmp_wav"
      ;;
  esac

  local args=("-m" "$model_file" "-f" "$wav_input")

  if [ -n "$lang_code" ]; then
    args+=("-l" "$lang_code")
  fi

  if [ -n "$out_prefix" ]; then
    out_dir="$(dirname "$out_prefix")"
    mkdir -p "$out_dir"
    args+=("-of" "$out_prefix")
  fi
  if [ "$want_txt" -eq 1 ]; then
    args+=("-otxt")
  fi
  if [ "$want_srt" -eq 1 ]; then
    args+=("-osrt")
  fi
  if [ "$want_vtt" -eq  1 ]; then
    args+=("-ovtt")
  fi

  "$BUILD_DIR/bin/whisper-cli" "${args[@]}"

  if [ -n "$tmp_wav" ] && [ -f "$tmp_wav" ]; then
    rm -f "$tmp_wav"
  fi
}

cmd="${1:-}"
case "$cmd" in
  install)
    install_whispercpp
    ;;
  install-deps)
    ensure_python_deps
    ;;
  list-models)
    list_models
    ;;
  download-model)
    shift || true
    download_model "${1:-$MODEL_NAME_DEFAULT}"
    ;;
  generate-coreml)
    shift || true
    generate_coreml_model "${1:-$MODEL_NAME_DEFAULT}"
    ;;
  convert)
    shift || true
    convert_media "${1:-}" "${2:-}"
    ;;
  transcribe)
    shift || true
    transcribe "$@"
    ;;
  -h|--help|help|"")
    usage
    ;;
  *)
    echo "Unknown command: $cmd" >&2
    usage
    exit 1
    ;;
esac
