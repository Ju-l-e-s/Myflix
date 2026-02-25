import os
import json
import subprocess
import hashlib
import time
import logging
from pathlib import Path

SHARE_DIR = (
    "/media_share/"
    if os.environ.get("DOCKER_MODE") == "true"
    else "/home/jules/media_share/"
)
CONFIG_FILE = (
    "/app/config_share.json"
    if os.environ.get("DOCKER_MODE") == "true"
    else "/home/jules/scripts/config_share.json"
)
BASE_URL = "https://share.juleslaconfourque.fr"

logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
)


def load_config():
    try:
        with open(CONFIG_FILE, "r") as f:
            return json.load(f)
    except:
        return {"max_gb": 4.0, "max_res": 1080}


def get_video_info(path):
    """Returns (size_gb, resolution_height)."""
    try:
        p = Path(path)
        if p.is_file():
            size = p.stat().st_size
        else:
            size = sum(f.stat().st_size for f in p.rglob("*") if f.is_file())

        size_gb = round(size / (1024**3), 2)

        # Get resolution via ffprobe
        probe_path = str(path)
        if p.is_dir():
            for f in p.rglob("*"):
                if f.suffix.lower() in (".mkv", ".mp4"):
                    probe_path = str(f)
                    break

        cmd = [
            "ffprobe",
            "-v",
            "error",
            "-select_streams",
            "v:0",
            "-show_entries",
            "stream=height",
            "-of",
            "csv=s=x:p=0",
            probe_path,
        ]
        res = subprocess.run(cmd, capture_output=True, text=True)
        height = int(res.stdout.strip()) if res.stdout.strip().isdigit() else 0
        return size_gb, height
    except Exception as e:
        logging.error(f"FFprobe error: {e}")
        return 0, 0


def remux_subtitles(video_path, sub_paths, output_path):
    """Remux video with subtitles using ffmpeg (softsub)."""
    try:
        cmd = ["ffmpeg", "-i", str(video_path)]
        for sub in sub_paths:
            cmd.extend(["-i", str(sub)])

        # Map all streams from all inputs
        cmd.extend(["-map", "0"])
        for i in range(1, len(sub_paths) + 1):
            cmd.extend(["-map", str(i)])

        # Stream copy video/audio, encode subs to srt if needed (mkv handles srt)
        cmd.extend(["-c", "copy", "-y", str(output_path)])

        logging.info(f"Running remux: {' '.join(cmd)}")
        res = subprocess.run(cmd, capture_output=True, text=True)
        if res.returncode != 0:
            logging.error(f"FFmpeg failed: {res.stderr}")
            return False
        return True
    except Exception as e:
        logging.error(f"Remux error: {e}")
        return False


def generate_secure_link(source_path, remux=False):
    config = load_config()
    os.makedirs(SHARE_DIR, exist_ok=True)

    src = Path(source_path)
    if not src.exists():
        logging.error(f"Source not found: {source_path}")
        return None

    # Validation Compliance
    size_gb, height = get_video_info(source_path)
    if size_gb > config.get("max_gb", 4.0) or height > config.get("max_res", 1080):
        logging.warning(f"Compliance Check Failed: {size_gb}GB, {height}p")
        return None

    # Hashing
    timestamp = str(int(time.time()))
    hash_object = hashlib.sha256(f"{src.name}{timestamp}".encode())
    unique_hash = hash_object.hexdigest()[:16]

    # Discovery Logic
    parent_dir = src.parent
    video_stem = src.stem
    subtitles = list(parent_dir.glob(f"{video_stem}*.srt"))

    # Case 1: Remux with Subtitles requested
    if remux and subtitles:
        # For remuxing, we use .mkv as output to ensure subtitle compatibility
        video_share_name = f"{unique_hash}_{src.stem}.mkv"
        video_dest_path = Path(SHARE_DIR) / video_share_name

        if remux_subtitles(src, subtitles, video_dest_path):
            return {"video": f"{BASE_URL}/{video_share_name}", "type": "remuxed"}
        else:
            logging.warning("Remux failed, falling back to symlinks.")

    # Case 2: Standard Symlinks (either requested or remux failed/no subtitles)
    video_share_name = f"{unique_hash}_{src.name}"
    video_dest_path = Path(SHARE_DIR) / video_share_name

    try:
        if video_dest_path.exists():
            video_dest_path.unlink()
        os.symlink(src, video_dest_path)
    except Exception as e:
        logging.error(f"Video symlink failed: {e}")
        return None

    result = {
        "video": f"{BASE_URL}/{video_share_name}",
        "subtitles": [],
        "type": "symlink",
    }

    # Symlink subtitles too
    for sub in subtitles:
        sub_share_name = f"{unique_hash}_{sub.name}"
        sub_dest_path = Path(SHARE_DIR) / sub_share_name
        try:
            if sub_dest_path.exists():
                sub_dest_path.unlink()
            os.symlink(sub, sub_dest_path)
            result["subtitles"].append(f"{BASE_URL}/{sub_share_name}")
        except Exception as e:
            logging.error(f"Subtitle symlink failed for {sub.name}: {e}")

    return result
