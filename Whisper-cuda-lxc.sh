#!/usr/bin/env bash
set -e

# Whisper CUDA LXC Installer (whisper.cpp with GPU acceleration)
# Author: Based on Whisper.sh with whisper.cpp integration
# License: MIT

source <(curl -s https://raw.githubusercontent.com/community-scripts/ProxmoxVE/main/misc/build.func)

APP="Whisper CUDA API"
HOSTNAME="whisper-cuda"
DISK_SIZE="40"
CORES="4"
MEMORY="8192"
BRIDGE="vmbr0"
OSVERSION="12"
PORT="8000"

header_info() {
  clear
  cat <<EOF
╔══════════════════════════════════════╗
║      Whisper CUDA LXC (whisper.cpp)   ║
║    GPU-accelerated whisper.cpp API   ║
╚══════════════════════════════════════╝
EOF
}

header_info

# ──────────────────────────────────────────────
# Detect NVIDIA GPU on host
# ──────────────────────────────────────────────
echo "Checking for NVIDIA GPU on host..."
GPU_DETECTED=0
GPU_NAME=""

if command -v nvidia-smi &>/dev/null && nvidia-smi &>/dev/null; then
  GPU_DETECTED=1
  GPU_NAME=$(nvidia-smi --query-gpu=name --format=csv,noheader | head -1)
  echo "NVIDIA GPU detected via nvidia-smi: $GPU_NAME"
elif lspci 2>/dev/null | grep -qi nvidia; then
  GPU_DETECTED=1
  GPU_NAME=$(lspci | grep -i nvidia | grep -i vga | head -1 | sed 's/.*: //')
  echo "NVIDIA GPU detected via lspci: $GPU_NAME"
else
  echo "No NVIDIA GPU detected — will use CPU mode"
fi

if [ "$GPU_DETECTED" -eq 1 ] && [ ! -e /dev/nvidia0 ]; then
  modprobe nvidia || true
  modprobe nvidia-uvm || true
  sleep 2
fi

# ──────────────────────────────────────────────
# Create LXC
# ──────────────────────────────────────────────
NEXTID=$(pvesh get /cluster/nextid)

mapfile -t STORAGE_OPTIONS < <(
  pvesm status -content rootdir 2>/dev/null | awk 'NR>1 {print $1}'
)

if [ ${#STORAGE_OPTIONS[@]} -eq 0 ]; then
  echo "ERROR: No container storage found"
  exit 1
fi

echo "Available storage:"
select STORAGE in "${STORAGE_OPTIONS[@]}"; do
  [[ -n "$STORAGE" ]] && break
done

echo "Using storage: $STORAGE"
pveam update

TEMPLATE=$(pveam available \
  | awk '$2 ~ /^debian-12-standard/ {print $2}' \
  | sort -V \
  | tail -n1)

if [ -z "$TEMPLATE" ]; then
  echo "ERROR: Debian template not found"
  exit 1
fi

pveam download local "$TEMPLATE"

pct create "$NEXTID" "local:vztmpl/$TEMPLATE" \
  --hostname "$HOSTNAME" \
  --cores "$CORES" \
  --memory "$MEMORY" \
  --rootfs "$STORAGE:$DISK_SIZE" \
  --net0 name=eth0,bridge="$BRIDGE",ip=dhcp \
  --unprivileged 0 \
  --features nesting=1 \
  --onboot 1

# ──────────────────────────────────────────────
# Configure GPU passthrough
# ──────────────────────────────────────────────
if [ "$GPU_DETECTED" -eq 1 ]; then
  CONFIG_FILE="/etc/pve/lxc/${NEXTID}.conf"
  
  # Add cgroup device access (correct major numbers)
  echo "lxc.cgroup2.devices.allow: c 195:* rwm" >> "$CONFIG_FILE"  # nvidia0, nvidiactl
  echo "lxc.cgroup2.devices.allow: c 511:* rwm" >> "$CONFIG_FILE"  # nvidia-uvm
  echo "lxc.cgroup2.devices.allow: c 236:* rwm" >> "$CONFIG_FILE"  # nvidia-caps
  echo "lxc.cgroup2.devices.allow: c 509:* rwm" >> "$CONFIG_FILE"  # nvidia-modeset
  
  # Mount NVIDIA devices
  echo "lxc.mount.entry: /dev/nvidia0 dev/nvidia0 none bind,optional,create=file" >> "$CONFIG_FILE"
  echo "lxc.mount.entry: /dev/nvidiactl dev/nvidiactl none bind,optional,create=file" >> "$CONFIG_FILE"
  echo "lxc.mount.entry: /dev/nvidia-uvm dev/nvidia-uvm none bind,optional,create=file" >> "$CONFIG_FILE"
  echo "lxc.mount.entry: /dev/nvidia-uvm-tools dev/nvidia-uvm-tools none bind,optional,create=file" >> "$CONFIG_FILE"
  echo "lxc.mount.entry: /dev/nvidia-modeset dev/nvidia-modeset none bind,optional,create=file" >> "$CONFIG_FILE"
  
  # Mount nvidia-caps devices
  echo "lxc.mount.entry: /dev/nvidia-caps/nvidia-cap1 dev/nvidia-caps/nvidia-cap1 none bind,optional,create=file" >> "$CONFIG_FILE"
  echo "lxc.mount.entry: /dev/nvidia-caps/nvidia-cap2 dev/nvidia-caps/nvidia-cap2 none bind,optional,create=file" >> "$CONFIG_FILE"
  
  echo "GPU passthrough configured"
fi

pct start "$NEXTID"
sleep 5

# ──────────────────────────────────────────────
# Install NVIDIA driver in container (if GPU detected)
# ──────────────────────────────────────────────
if [ "$GPU_DETECTED" -eq 1 ]; then
  echo "Installing NVIDIA driver in container..."
  
  # Find NVIDIA driver .run file on host
  NVIDIA_DRIVER_RUN=$(find /root /tmp /usr/local/src -name "NVIDIA-Linux-x86_64-*.run" 2>/dev/null | head -n1)
  
  if [ -n "$NVIDIA_DRIVER_RUN" ]; then
    echo "Found NVIDIA driver: $NVIDIA_DRIVER_RUN"
    
    # Push driver to container
    pct push "$NEXTID" "$NVIDIA_DRIVER_RUN" /root/nvidia-driver.run
    
    # Install driver in container (no kernel modules - uses host kernel)
    echo "Installing driver in container (this may take a few minutes)..."
    pct exec "$NEXTID" -- bash -c "chmod +x /root/nvidia-driver.run && /root/nvidia-driver.run --no-kernel-modules --silent" || {
      echo "WARNING: Driver installation had issues, continuing anyway..."
    }
  else
    echo "WARNING: No NVIDIA driver .run file found on host"
    echo "You can install CUDA toolkit directly in container instead"
  fi
fi

# ──────────────────────────────────────────────
# Install whisper.cpp with CUDA support
# ──────────────────────────────────────────────
pct exec "$NEXTID" -- bash -c "GPU_MODE=$GPU_DETECTED bash -s" <<'EOF'
set -e

apt update && apt upgrade -y
apt install -y git cmake build-essential ffmpeg pkg-config curl ca-certificates pciutils

# GPU detection inside container
USE_GPU=0
DEVICE_TYPE="cpu"

if command -v nvidia-smi &>/dev/null && nvidia-smi &>/dev/null; then
  USE_GPU=1
  DEVICE_TYPE="cuda"
  echo "GPU detected in container"
fi

# Install CUDA toolkit if GPU detected
if [ "$USE_GPU" -eq 1 ]; then
  echo "Installing CUDA toolkit from NVIDIA repository..."
  
  # Add NVIDIA CUDA repository
  wget https://developer.download.nvidia.com/compute/cuda/repos/debian12/x86_64/cuda-keyring_1.1-1_all.deb
  dpkg -i cuda-keyring_1.1-1_all.deb
  apt-get update
  
  # Install CUDA toolkit
  apt-get install -y cuda-toolkit-12-3
  
  # Add CUDA to PATH
  echo 'export PATH=/usr/local/cuda/bin:$PATH' >> /etc/environment
  echo 'export LD_LIBRARY_PATH=/usr/local/cuda/lib64:$LD_LIBRARY_PATH' >> /etc/environment
  
  # Verify CUDA installation
  if command -v nvcc >/dev/null 2>&1; then
    echo "CUDA version: $(nvcc --version | grep "release" | awk '{print $6}' | cut -c2-)"
  else
    echo "WARNING: nvcc not found after installation"
  fi
fi

# Build whisper.cpp with CUDA support
mkdir -p /opt/whisper-cuda
cd /opt/whisper-cuda

if [ ! -d whisper.cpp ]; then
  git clone https://github.com/ggml-org/whisper.cpp.git
else
  cd whisper.cpp && git pull && cd ..
fi

cd whisper.cpp

# Build with CUDA if available, otherwise CPU
if [ "$USE_GPU" -eq 1 ] && command -v nvcc >/dev/null 2>&1; then
  echo "Building whisper.cpp with CUDA support..."
  # Load CUDA environment
  export PATH=/usr/local/cuda/bin:$PATH
  export LD_LIBRARY_PATH=/usr/local/cuda/lib64:$LD_LIBRARY_PATH
  cmake -S . -B build -DGGML_CUDA=ON
else
  echo "Building whisper.cpp (CPU-only)..."
  cmake -S . -B build
fi

cmake --build build -j 1 --config Release

# Download default model
echo "Downloading base.en model..."
sh ./models/download-ggml-model.sh base.en

# Save device type
echo "$DEVICE_TYPE" > /opt/whisper-cuda/device_type.txt

echo "whisper.cpp installation complete"
EOF

# ──────────────────────────────────────────────
# Create FastAPI wrapper for whisper-cli
# ──────────────────────────────────────────────
pct exec "$NEXTID" -- bash -c '
mkdir -p /opt/whisper-cuda/api
cd /opt/whisper-cuda/api

# Install Python for API server
apt install -y python3 python3-pip python3-venv
python3 -m venv venv
source venv/bin/activate
pip install --upgrade pip setuptools wheel
pip install fastapi uvicorn python-multipart

# Generate API key
openssl rand -hex 16 > /opt/whisper-cuda/api_key.txt
chmod 600 /opt/whisper-cuda/api_key.txt

# Create FastAPI app that wraps whisper-cli
cat > app.py <<PY
import subprocess, tempfile, os, json
from fastapi import FastAPI, UploadFile, File, HTTPException, Header, Query
from fastapi.responses import PlainTextResponse, StreamingResponse

API_KEY_FILE = "/opt/whisper-cuda/api_key.txt"
with open(API_KEY_FILE) as f: API_KEY=f.read().strip()

WHISPER_CLI = "/opt/whisper-cuda/whisper.cpp/build/bin/whisper-cli"
MODEL_PATH = "/opt/whisper-cuda/whisper.cpp/models/ggml-base.en.bin"
MODELS_DIR = "/opt/whisper-cuda/whisper.cpp/models"

app = FastAPI(title="Whisper CUDA API")
LAST_LOGS = ""

def check_api_key(key: str):
    if key != API_KEY:
        raise HTTPException(status_code=401, detail="Invalid API key")

def run_cmd(cmd):
    result = subprocess.run(cmd, capture_output=True, text=True)
    stdout = result.stdout or ""
    stderr = result.stderr or ""
    combined = (stdout + "\n" + stderr).strip()
    return result.returncode, stdout, stderr, combined

def sse_payload(payload):
    return f"data: {json.dumps(payload)}\n\n"

def resolve_model(model_name: str | None):
    if not model_name:
        return MODEL_PATH
    name = model_name.strip()
    if not name:
        return MODEL_PATH
    if not name.endswith(".bin"):
        name = f"ggml-{name}.bin"
    candidate = os.path.join(MODELS_DIR, name)
    return candidate if os.path.exists(candidate) else MODEL_PATH

def run_whisper_cli(audio_path, output_format="json", model_name=None, threads=None, beam_size=None, best_of=None, language=None):
    output_base = os.path.join("/tmp", os.path.basename(audio_path))
    model_path = resolve_model(model_name)
    cmd = [WHISPER_CLI, "-m", model_path, "-f", audio_path, "-of", output_base]
    if threads:
        cmd.extend(["-t", str(threads)])
    if beam_size:
        cmd.extend(["-bs", str(beam_size)])
    if best_of:
        cmd.extend(["-bo", str(best_of)])
    if language:
        cmd.extend(["-l", language])
    
    if output_format == "txt":
        cmd.extend(["-otxt"])
        code, stdout, stderr, combined = run_cmd(cmd)
        return {"text": stdout.strip(), "_log": combined, "_code": code}
    elif output_format == "srt":
        cmd.extend(["-osrt"])
        code, stdout, stderr, combined = run_cmd(cmd)
        srt_file = f"{output_base}.srt"
        if os.path.exists(srt_file):
            with open(srt_file, "r") as f:
                srt_content = f.read()
            os.unlink(srt_file)
            return {"content": srt_content, "_log": combined, "_code": code}
        return {"content": stdout, "_log": combined, "_code": code}
    elif output_format == "vtt":
        cmd.extend(["-ovtt"])
        code, stdout, stderr, combined = run_cmd(cmd)
        vtt_file = f"{output_base}.vtt"
        if os.path.exists(vtt_file):
            with open(vtt_file, "r") as f:
                vtt_content = f.read()
            os.unlink(vtt_file)
            return {"content": vtt_content, "_log": combined, "_code": code}
        return {"content": stdout, "_log": combined, "_code": code}
    else:
        # JSON format - need to parse from whisper output
        code, stdout, stderr, combined = run_cmd(cmd)
        # whisper.cpp outputs plain text by default
        return {"text": stdout.strip(), "language": "en", "_log": combined, "_code": code}

@app.post("/transcribe")
async def transcribe(
    file: UploadFile = File(...),
    x_api_key: str = Header(...),
    debug: bool = Query(False),
    model: str | None = Query(None),
    threads: int | None = Query(None),
    beam_size: int | None = Query(None),
    best_of: int | None = Query(None),
    language: str | None = Query(None),
):
    check_api_key(x_api_key)
    global LAST_LOGS
    with tempfile.NamedTemporaryFile(delete=False, suffix=os.path.splitext(file.filename or "")[1]) as tmp:
        tmp.write(await file.read())
        tmp.flush()
        with tempfile.NamedTemporaryFile(delete=False, suffix=".wav") as wav_tmp:
            ffmpeg_cmd = ["ffmpeg", "-y", "-i", tmp.name, "-ar", "16000", "-ac", "1", "-c:a", "pcm_s16le", wav_tmp.name]
            _, _, _, ffmpeg_log = run_cmd(ffmpeg_cmd)
            result = run_whisper_cli(wav_tmp.name, "json", model, threads, beam_size, best_of, language)
            LAST_LOGS = (ffmpeg_log + "\n\n" + result.get("_log", "")).strip()
            os.unlink(wav_tmp.name)
        os.unlink(tmp.name)
    if debug:
        result["_log"] = LAST_LOGS
    return result

@app.post("/transcribe/txt")
async def transcribe_txt(
    file: UploadFile = File(...),
    x_api_key: str = Header(...),
    debug: bool = Query(False),
    model: str | None = Query(None),
    threads: int | None = Query(None),
    beam_size: int | None = Query(None),
    best_of: int | None = Query(None),
    language: str | None = Query(None),
):
    check_api_key(x_api_key)
    global LAST_LOGS
    with tempfile.NamedTemporaryFile(delete=False, suffix=os.path.splitext(file.filename or "")[1]) as tmp:
        tmp.write(await file.read())
        tmp.flush()
        with tempfile.NamedTemporaryFile(delete=False, suffix=".wav") as wav_tmp:
            ffmpeg_cmd = ["ffmpeg", "-y", "-i", tmp.name, "-ar", "16000", "-ac", "1", "-c:a", "pcm_s16le", wav_tmp.name]
            _, _, _, ffmpeg_log = run_cmd(ffmpeg_cmd)
            result = run_whisper_cli(wav_tmp.name, "txt", model, threads, beam_size, best_of, language)
            LAST_LOGS = (ffmpeg_log + "\n\n" + result.get("_log", "")).strip()
            os.unlink(wav_tmp.name)
        os.unlink(tmp.name)
    if debug:
        result["_log"] = LAST_LOGS
    return result

@app.post("/transcribe/srt")
async def transcribe_srt(
    file: UploadFile = File(...),
    x_api_key: str = Header(...),
    debug: bool = Query(False),
    model: str | None = Query(None),
    threads: int | None = Query(None),
    beam_size: int | None = Query(None),
    best_of: int | None = Query(None),
    language: str | None = Query(None),
):
    check_api_key(x_api_key)
    global LAST_LOGS
    with tempfile.NamedTemporaryFile(delete=False, suffix=os.path.splitext(file.filename or "")[1]) as tmp:
        tmp.write(await file.read())
        tmp.flush()
        with tempfile.NamedTemporaryFile(delete=False, suffix=".wav") as wav_tmp:
            ffmpeg_cmd = ["ffmpeg", "-y", "-i", tmp.name, "-ar", "16000", "-ac", "1", "-c:a", "pcm_s16le", wav_tmp.name]
            _, _, _, ffmpeg_log = run_cmd(ffmpeg_cmd)
            result = run_whisper_cli(wav_tmp.name, "srt", model, threads, beam_size, best_of, language)
            LAST_LOGS = (ffmpeg_log + "\n\n" + result.get("_log", "")).strip()
            os.unlink(wav_tmp.name)
        os.unlink(tmp.name)
    if debug:
        return {"srt": result.get("content", ""), "_log": LAST_LOGS}
    return PlainTextResponse(content=result.get("content", ""), media_type="text/plain")

@app.post("/transcribe/vtt")
async def transcribe_vtt(
    file: UploadFile = File(...),
    x_api_key: str = Header(...),
    debug: bool = Query(False),
    model: str | None = Query(None),
    threads: int | None = Query(None),
    beam_size: int | None = Query(None),
    best_of: int | None = Query(None),
    language: str | None = Query(None),
):
    check_api_key(x_api_key)
    global LAST_LOGS
    with tempfile.NamedTemporaryFile(delete=False, suffix=os.path.splitext(file.filename or "")[1]) as tmp:
        tmp.write(await file.read())
        tmp.flush()
        with tempfile.NamedTemporaryFile(delete=False, suffix=".wav") as wav_tmp:
            ffmpeg_cmd = ["ffmpeg", "-y", "-i", tmp.name, "-ar", "16000", "-ac", "1", "-c:a", "pcm_s16le", wav_tmp.name]
            _, _, _, ffmpeg_log = run_cmd(ffmpeg_cmd)
            result = run_whisper_cli(wav_tmp.name, "vtt", model, threads, beam_size, best_of, language)
            LAST_LOGS = (ffmpeg_log + "\n\n" + result.get("_log", "")).strip()
            os.unlink(wav_tmp.name)
        os.unlink(tmp.name)
    if debug:
        return {"vtt": result.get("content", ""), "_log": LAST_LOGS}
    return PlainTextResponse(content=result.get("content", ""), media_type="text/vtt")

@app.post("/transcribe/stream/{fmt}")
async def transcribe_stream(
    fmt: str,
    file: UploadFile = File(...),
    x_api_key: str = Header(...),
    model: str | None = Query(None),
    threads: int | None = Query(None),
    beam_size: int | None = Query(None),
    best_of: int | None = Query(None),
    language: str | None = Query(None),
):
    check_api_key(x_api_key)
    if fmt not in ("srt", "vtt", "txt"):
        raise HTTPException(status_code=400, detail="Unsupported format")

    raw_tmp = tempfile.NamedTemporaryFile(delete=False, suffix=os.path.splitext(file.filename or "")[1])
    raw_tmp.write(await file.read())
    raw_tmp.flush()
    raw_tmp.close()
    wav_tmp = tempfile.NamedTemporaryFile(delete=False, suffix=".wav")
    wav_tmp.close()

    def event_stream():
        global LAST_LOGS
        logs = []
        try:
            ffmpeg_cmd = ["ffmpeg", "-y", "-i", raw_tmp.name, "-ar", "16000", "-ac", "1", "-c:a", "pcm_s16le", wav_tmp.name]
            logs.append("Running ffmpeg: " + " ".join(ffmpeg_cmd))
            yield sse_payload({"type": "log", "message": logs[-1]})

            ffmpeg_proc = subprocess.Popen(ffmpeg_cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True, bufsize=1)
            for line in ffmpeg_proc.stdout:
                line = line.rstrip()
                if line:
                    logs.append(line)
                    yield sse_payload({"type": "log", "message": line})
            ffmpeg_code = ffmpeg_proc.wait()
            if ffmpeg_code != 0:
                err = f"ffmpeg failed with code {ffmpeg_code}"
                logs.append(err)
                yield sse_payload({"type": "error", "message": err})
                LAST_LOGS = "\n".join(logs)
                return

            output_base = os.path.join("/tmp", os.path.basename(wav_tmp.name))
            model_path = resolve_model(model)
            whisper_cmd = [WHISPER_CLI, "-m", model_path, "-f", wav_tmp.name, "-of", output_base]
            if threads:
                whisper_cmd.extend(["-t", str(threads)])
            if beam_size:
                whisper_cmd.extend(["-bs", str(beam_size)])
            if best_of:
                whisper_cmd.extend(["-bo", str(best_of)])
            if language:
                whisper_cmd.extend(["-l", language])
            if fmt == "txt":
                whisper_cmd.append("-otxt")
            elif fmt == "srt":
                whisper_cmd.append("-osrt")
            elif fmt == "vtt":
                whisper_cmd.append("-ovtt")

            logs.append("Running whisper-cli: " + " ".join(whisper_cmd))
            yield sse_payload({"type": "log", "message": logs[-1]})
            whisper_proc = subprocess.Popen(whisper_cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True, bufsize=1)
            for line in whisper_proc.stdout:
                line = line.rstrip()
                if line:
                    logs.append(line)
                    yield sse_payload({"type": "log", "message": line})
            whisper_code = whisper_proc.wait()
            if whisper_code != 0:
                err = f"whisper-cli failed with code {whisper_code}"
                logs.append(err)
                yield sse_payload({"type": "error", "message": err})
                LAST_LOGS = "\n".join(logs)
                return

            content = ""
            if fmt == "txt":
                txt_file = output_base + ".txt"
                if os.path.exists(txt_file):
                    with open(txt_file, "r") as f:
                        content = f.read()
                    os.unlink(txt_file)
            elif fmt == "srt":
                srt_file = output_base + ".srt"
                if os.path.exists(srt_file):
                    with open(srt_file, "r") as f:
                        content = f.read()
                    os.unlink(srt_file)
            elif fmt == "vtt":
                vtt_file = output_base + ".vtt"
                if os.path.exists(vtt_file):
                    with open(vtt_file, "r") as f:
                        content = f.read()
                    os.unlink(vtt_file)

            LAST_LOGS = "\n".join(logs)
            yield sse_payload({"type": "result", "format": fmt, "content": content})
        finally:
            if os.path.exists(raw_tmp.name):
                os.unlink(raw_tmp.name)
            if os.path.exists(wav_tmp.name):
                os.unlink(wav_tmp.name)

    return StreamingResponse(event_stream(), media_type="text/event-stream")

@app.get("/debug/last")
async def debug_last(x_api_key: str = Header(...)):
    check_api_key(x_api_key)
    return {"log": LAST_LOGS}

@app.get("/models")
async def list_models(x_api_key: str = Header(...)):
    check_api_key(x_api_key)
    try:
        models = sorted([f for f in os.listdir(MODELS_DIR) if f.endswith(".bin")])
    except FileNotFoundError:
        models = []
    return {"models": models, "default": os.path.basename(MODEL_PATH)}
PY

# Install ffmpeg for audio conversion
apt install -y ffmpeg
'

# ──────────────────────────────────────────────
# Create systemd service
# ──────────────────────────────────────────────
pct exec "$NEXTID" -- bash -c '
cat > /etc/systemd/system/whisper-cuda-api.service <<SERVICE
[Unit]
Description=Whisper CUDA API
After=network.target

[Service]
WorkingDirectory=/opt/whisper-cuda/api
Environment="PATH=/opt/whisper-cuda/api/venv/bin:/usr/local/bin:/usr/bin:/bin"
ExecStart=/opt/whisper-cuda/api/venv/bin/uvicorn app:app --host 0.0.0.0 --port 8000

Restart=always
[Install]
WantedBy=multi-user.target
SERVICE

systemctl daemon-reload
systemctl enable --now whisper-cuda-api.service
'

# ──────────────────────────────────────────────
# Display results
# ──────────────────────────────────────────────
IP=$(pct exec "$NEXTID" -- hostname -I | awk '{print $1}')
DEVICE_TYPE=$(pct exec "$NEXTID" -- cat /opt/whisper-cuda/device_type.txt 2>/dev/null || echo "cpu")
API_KEY=$(pct exec "$NEXTID" -- cat /opt/whisper-cuda/api_key.txt)

echo ""
echo "══════════════════════════════════════════════════"
echo " Whisper CUDA API is ready"
echo " URL: http://$IP:8000"
echo " Device: $DEVICE_TYPE ($([ "$DEVICE_TYPE" = "cuda" ] && echo "GPU" || echo "CPU"))"
echo " Engine: whisper.cpp (C++ with CUDA support)"
echo " API key: $API_KEY"
echo " Endpoints:"
echo "   POST /transcribe      → JSON"
echo "   POST /transcribe/txt  → Plain text"
echo "   POST /transcribe/srt  → SRT"
echo "   POST /transcribe/vtt  → VTT"
echo "══════════════════════════════════════════════════"
echo ""
echo "To test:"
echo "  curl -X POST -H 'x-api-key: $API_KEY' -F 'file=@audio.wav' http://$IP:8000/transcribe"
