#!/usr/bin/env bash
set -e

# Copyright (c) 2026
# Author: <your-name>
# License: MIT
# LibreTranslate LXC with GPU support (NVIDIA) and CPU fallback

source <(curl -s https://raw.githubusercontent.com/community-scripts/ProxmoxVE/main/misc/build.func)

APP="LibreTranslate"
HOSTNAME="libretranslate"
DISK_SIZE="40"
CORES="4"
MEMORY="8192"
BRIDGE="vmbr0"
OSVERSION="12"
PORT="5000"

header_info() {
  clear
  cat <<EOF
╔══════════════════════════════════════╗
║     LibreTranslate LXC (GPU/CPU)     ║
║        Debian 12 (Native)            ║
╚══════════════════════════════════════╝
EOF
}

header_info

# ─────────────────────────────────────────────────────────────
# Step 1: Detect NVIDIA GPU on the Proxmox host
# ─────────────────────────────────────────────────────────────
echo "Checking for NVIDIA GPU on host..."
GPU_DETECTED=0
GPU_NAME=""

# Primary detection: nvidia-smi (most reliable if drivers installed)
if command -v nvidia-smi &>/dev/null && nvidia-smi &>/dev/null; then
  GPU_DETECTED=1
  GPU_NAME=$(nvidia-smi --query-gpu=name --format=csv,noheader | head -1)
  echo "NVIDIA GPU detected via nvidia-smi: $GPU_NAME"
# Fallback: lspci
elif lspci 2>/dev/null | grep -qi nvidia; then
  GPU_DETECTED=1
  GPU_NAME=$(lspci | grep -i nvidia | grep -i vga | head -1 | sed 's/.*: //')
  echo "NVIDIA GPU detected via lspci: $GPU_NAME"
else
  echo "No NVIDIA GPU detected — will use CPU-only mode"
fi

# Verify /dev/nvidia* devices exist (required for passthrough)
if [ "$GPU_DETECTED" -eq 1 ]; then
  if [ ! -e /dev/nvidia0 ]; then
    echo "WARNING: GPU detected but /dev/nvidia0 not found"
    echo "Loading NVIDIA kernel modules..."
    modprobe nvidia || true
    modprobe nvidia-uvm || true
    sleep 2
  fi
  
  if [ -e /dev/nvidia0 ]; then
    echo "NVIDIA device nodes found: $(ls /dev/nvidia* 2>/dev/null | tr '\n' ' ')"
  else
    echo "ERROR: /dev/nvidia* devices still not available"
    echo "GPU passthrough will not work. Falling back to CPU mode."
    GPU_DETECTED=0
  fi
fi

NEXTID=$(pvesh get /cluster/nextid)

echo "Detecting available storage..."
mapfile -t STORAGE_OPTIONS < <(
  pvesm status -content rootdir 2>/dev/null | awk 'NR>1 {print $1}'
)

if [ ${#STORAGE_OPTIONS[@]} -eq 0 ]; then
  echo "ERROR: No container storage found"
  pvesm status
  exit 1
fi

echo "Available storage:"
select STORAGE in "${STORAGE_OPTIONS[@]}"; do
  [[ -n "$STORAGE" ]] && break
done

echo "Using storage: $STORAGE"
echo "Creating LXC $NEXTID..."

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

# ─────────────────────────────────────────────────────────────
# Step 2: Create LXC container
# ─────────────────────────────────────────────────────────────
pct create "$NEXTID" "local:vztmpl/$TEMPLATE" \
  --hostname "$HOSTNAME" \
  --cores "$CORES" \
  --memory "$MEMORY" \
  --rootfs "$STORAGE:$DISK_SIZE" \
  --net0 name=eth0,bridge="$BRIDGE",ip=dhcp \
  --unprivileged 0 \
  --features nesting=1 \
  --onboot 1

# ─────────────────────────────────────────────────────────────
# Step 3: Configure GPU passthrough if GPU detected
# ─────────────────────────────────────────────────────────────
if [ "$GPU_DETECTED" -eq 1 ]; then
  echo "Configuring GPU passthrough for container $NEXTID..."
  
  CONFIG_FILE="/etc/pve/lxc/${NEXTID}.conf"
  
  # Add cgroup device access
  echo "lxc.cgroup2.devices.allow: c 195:* rwm" >> "$CONFIG_FILE"  # nvidia0, nvidiactl
  echo "lxc.cgroup2.devices.allow: c 234:* rwm" >> "$CONFIG_FILE"  # nvidia-uvm, nvidia-uvm-tools
  echo "lxc.cgroup2.devices.allow: c 237:* rwm" >> "$CONFIG_FILE"  # nvidia-caps
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

# Start the container
pct start "$NEXTID"
sleep 5

# ─────────────────────────────────────────────────────────────
# Step 3b: Install NVIDIA driver in container (if GPU detected)
# ─────────────────────────────────────────────────────────────
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
    # No .run file found - try to get driver version from host
    DRIVER_VERSION=$(nvidia-smi --query-gpu=driver_version --format=csv,noheader 2>/dev/null | head -n1)
    echo "WARNING: No NVIDIA driver .run file found on host"
    echo "Host driver version: $DRIVER_VERSION"
    echo "You can download it from: https://www.nvidia.com/Download/index.aspx"
    echo "Place the .run file in /root/ and re-run, or install manually later"
  fi
  
  # Install NVIDIA Container Toolkit
  echo "Installing NVIDIA Container Toolkit..."
  pct exec "$NEXTID" -- bash -c '
    apt-get install -y gpg curl
    curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg
    curl -s -L https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list | \
      sed "s#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g" | \
      tee /etc/apt/sources.list.d/nvidia-container-toolkit.list
    apt-get update
    apt-get install -y nvidia-container-toolkit
  ' || echo "WARNING: NVIDIA Container Toolkit installation had issues"
  
  # Configure NVIDIA Container Runtime
  pct exec "$NEXTID" -- bash -c '
    if [ -f /etc/nvidia-container-runtime/config.toml ]; then
      sed -i "s/#no-cgroups = false/no-cgroups = true/" /etc/nvidia-container-runtime/config.toml
      sed -i "s/no-cgroups = false/no-cgroups = true/" /etc/nvidia-container-runtime/config.toml
      echo "Configured no-cgroups = true"
    fi
  '
  
  echo "NVIDIA setup complete"
fi

# ─────────────────────────────────────────────────────────────
# Step 4: Install LibreTranslate inside the container
# ─────────────────────────────────────────────────────────────
pct exec "$NEXTID" -- bash -c "GPU_MODE=$GPU_DETECTED bash -s" <<'EOF'
set -e

echo "Updating system..."
apt update && apt upgrade -y

echo "Installing base dependencies..."
apt install -y \
  python3 \
  python3-venv \
  python3-pip \
  git \
  build-essential \
  pkg-config \
  cmake \
  libsentencepiece-dev \
  curl \
  ca-certificates \
  pciutils \
  wget \
  unzip

# ─────────────────────────────────────────────────────────────
# Detect GPU inside container
# ─────────────────────────────────────────────────────────────
USE_GPU=0
DEVICE_TYPE="cpu"

# Check if nvidia-smi works (driver installed and GPU accessible)
if command -v nvidia-smi &>/dev/null && nvidia-smi &>/dev/null; then
  echo "NVIDIA GPU available in container:"
  nvidia-smi --query-gpu=name,driver_version,memory.total --format=csv,noheader
  USE_GPU=1
  DEVICE_TYPE="cuda"
elif [ -e /dev/nvidia0 ]; then
  echo "NVIDIA device found but driver not working — using CPU mode"
else
  echo "No GPU device found — using CPU mode"
fi

mkdir -p /opt/libretranslate
cd /opt/libretranslate

git clone https://github.com/LibreTranslate/LibreTranslate.git .

python3 -m venv venv
source venv/bin/activate

pip install --upgrade pip setuptools wheel

# ─────────────────────────────────────────────────────────────
# Install PyTorch (GPU or CPU version)
# ─────────────────────────────────────────────────────────────
if [ "$USE_GPU" -eq 1 ]; then
  echo "Installing PyTorch with CUDA support..."
  
  # Detect CUDA version from nvidia-smi
  CUDA_VERSION=$(nvidia-smi 2>/dev/null | grep -oP 'CUDA Version: \K[0-9]+\.[0-9]+' || echo "")
  CUDA_MAJOR=$(echo "$CUDA_VERSION" | cut -d. -f1)
  
  echo "Detected CUDA version: $CUDA_VERSION"
  
  # LibreTranslate 1.8.x requires torch==2.4.0
  # Use cu121 which is compatible with torch 2.4.0 and CUDA 12.x drivers
  echo "Installing PyTorch 2.4.0 with CUDA 12.1 support (LibreTranslate compatible)"
  pip install torch==2.4.0 torchvision==0.19.0 torchaudio==2.4.0 --index-url https://download.pytorch.org/whl/cu121
else
  echo "Installing PyTorch CPU version..."
  pip install torch torchvision torchaudio --index-url https://download.pytorch.org/whl/cpu
fi

# Install LibreTranslate
pip install .

# Install Argos Translate and support packages
pip install argostranslate argos-translate-files translatehtml

# ─────────────────────────────────────────────────────────────
# Script to download Argos models
# ─────────────────────────────────────────────────────────────
cat >/opt/libretranslate/install_argos_models.py <<PY
#!/usr/bin/env python3
from argostranslate import package, translate

print("Updating Argos Translate package index...")
package.update_package_index()

available = package.get_available_packages()
print(f"Found {len(available)} available packages")

pairs = [
    ("en", "ru"), ("ru", "en"),
    ("en", "pl"), ("pl", "en"),
    ("en", "es"), ("es", "en"),
    ("en", "fr"), ("fr", "en"),
    ("en", "de"), ("de", "en"),
    ("en", "it"), ("it", "en"),
    ("en", "pt"), ("pt", "en"),
    ("en", "nl"), ("nl", "en"),
]

for from_code, to_code in pairs:
    pkg = next((p for p in available if p.from_code == from_code and p.to_code == to_code), None)
    if not pkg:
        print(f"[!] Model {from_code}->{to_code} not found")
        continue
    print(f"Installing {pkg.from_name} -> {pkg.to_name}...")
    try:
        download_path = pkg.download()
        package.install_from_path(download_path)
        print(f"  Installed successfully")
    except Exception as e:
        print(f"  [!] Failed: {e}")

print("Installed languages:", [str(lang) for lang in translate.get_installed_languages()])
PY

python3 /opt/libretranslate/install_argos_models.py || echo "[!] Some Argos models failed"

# ─────────────────────────────────────────────────────────────
# Generate API key
# ─────────────────────────────────────────────────────────────
API_KEY=$(openssl rand -hex 16)
echo "$API_KEY" >/root/libretranslate-api-key.txt
chmod 600 /root/libretranslate-api-key.txt

# ─────────────────────────────────────────────────────────────
# Setup systemd service with GPU/CPU detection
# ─────────────────────────────────────────────────────────────
cat >/etc/systemd/system/libretranslate.service <<SERVICE
[Unit]
Description=LibreTranslate API
After=network.target

[Service]
WorkingDirectory=/opt/libretranslate
Environment="PATH=/opt/libretranslate/venv/bin"
Environment="ARGOS_DEVICE_TYPE=$DEVICE_TYPE"
ExecStart=/opt/libretranslate/venv/bin/libretranslate --host 0.0.0.0 --port 5000 --api-keys --req-limit 60 --req-limit-storage memory://

Restart=always

[Install]
WantedBy=multi-user.target
SERVICE

# Save device type for reference
echo "$DEVICE_TYPE" > /opt/libretranslate/device_type.txt

systemctl daemon-reload
systemctl enable --now libretranslate
EOF

# ─────────────────────────────────────────────────────────────
# Step 4b: Install language model helper script (base64 encoded to avoid heredoc issues)
# ─────────────────────────────────────────────────────────────
HELPER_SCRIPT_B64=$(cat <<'HELPER_SCRIPT_RAW' | base64
#!/usr/bin/env bash
set -euo pipefail

VENV_PY="/opt/libretranslate/venv/bin/python"
if [ ! -x "$VENV_PY" ]; then
  echo "LibreTranslate venv not found. Is LibreTranslate installed?" >&2
  exit 1
fi

cmd="${1:-}"
case "$cmd" in
  list-models)
    "$VENV_PY" -c '
from argostranslate import package
package.update_package_index()
for p in package.get_available_packages():
    print(f"{p.from_code}->{p.to_code}\t{p.from_name} -> {p.to_name}")
'
    ;;
  list-installed)
    "$VENV_PY" -c '
from argostranslate import translate
langs = translate.get_installed_languages()
if not langs:
    print("No languages installed")
else:
    for l in langs:
        print(f"{l.code}\t{l.name}")
'
    ;;
  install-model)
    from_code="${2:-}"
    to_code="${3:-}"
    if [ -z "$from_code" ] || [ -z "$to_code" ]; then
      echo "Usage: libretranslate-model install-model <from> <to>" >&2
      exit 1
    fi
    "$VENV_PY" -c "
from argostranslate import package
package.update_package_index()
pkg = next((p for p in package.get_available_packages() if p.from_code == '$from_code' and p.to_code == '$to_code'), None)
if not pkg:
    raise SystemExit('Model $from_code->$to_code not found. Run: libretranslate-model list-models')
print(f'Installing {pkg.from_name} -> {pkg.to_name}')
package.install_from_path(pkg.download())
print('Installed successfully')
"
    ;;
  -h|--help|help|"")
    echo "Usage:"
    echo "  libretranslate-model list-models"
    echo "  libretranslate-model list-installed"
    echo "  libretranslate-model install-model <from> <to>"
    echo ""
    echo "Examples:"
    echo "  libretranslate-model install-model en fr"
    echo "  libretranslate-model install-model fr en"
    ;;
  *)
    echo "Unknown command: $cmd" >&2
    exit 1
    ;;
esac
HELPER_SCRIPT_RAW
)
pct exec "$NEXTID" -- bash -c "echo '$HELPER_SCRIPT_B64' | base64 -d > /usr/local/bin/libretranslate-model && chmod +x /usr/local/bin/libretranslate-model"

# ─────────────────────────────────────────────────────────────
# Step 5: Display results
# ─────────────────────────────────────────────────────────────
IP=$(pct exec "$NEXTID" -- hostname -I | awk '{print $1}')
DEVICE_TYPE=$(pct exec "$NEXTID" -- cat /opt/libretranslate/device_type.txt 2>/dev/null || echo "cpu")

echo ""
echo "══════════════════════════════════════════════════"
echo " LibreTranslate is ready"
echo " URL: http://$IP:5000"
echo " Device: $DEVICE_TYPE ($([ "$DEVICE_TYPE" = "cuda" ] && echo "GPU" || echo "CPU"))"
echo " API key stored at:"
echo "   /root/libretranslate-api-key.txt"
echo " Language models helper inside container:"
echo "   libretranslate-model list-models"
echo "   libretranslate-model list-installed"
echo "   libretranslate-model install-model en fr"
echo "══════════════════════════════════════════════════"

if [ "$GPU_DETECTED" -eq 1 ]; then
  echo ""
  echo " GPU Notes:"
  echo "   - NVIDIA drivers must be installed on Proxmox host"
  echo "   - Run 'nvidia-smi' on host to verify driver"
  echo "   - Container uses host GPU via passthrough"
  echo ""
fi