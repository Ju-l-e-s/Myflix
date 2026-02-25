import os
import json
import subprocess
import hashlib
import time
import logging
from pathlib import Path
from config import DOCKER_MODE

BASE_DIR = os.path.dirname(os.path.abspath(__file__))
HOME = os.path.expanduser("~")
SHARE_DIR = "/media_share/" if DOCKER_MODE else os.path.join(HOME, "media_share/")
CONFIG_FILE = "/app/config_share.json" if DOCKER_MODE else os.path.join(BASE_DIR, "config_share.json")
BASE_URL = "https://share.juleslaconfourque.fr"

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')

def load_config():
    try:
        with open(CONFIG_FILE, 'r') as f: return json.load(f)
    except: return {"max_gb": 4.0, "max_res": 1080}

def get_video_info(path):
    """Returns (size_gb, resolution_height)."""
    try:
        p = Path(path)
        if p.is_file():
            size = p.stat().st_size
        else:
            size = sum(f.stat().st_size for f in p.rglob('*') if f.is_file())
        
        size_gb = round(size / (1024**3), 2)
        
        # Get resolution via ffprobe
        probe_path = str(path)
        if p.is_dir():
            for f in p.rglob('*'):
                if f.suffix.lower() in ('.mkv', '.mp4', '.m2ts', '.avi', '.ts'):
                    probe_path = str(f)
                    break
        
        cmd = ["ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=height", "-of", "csv=s=x:p=0", probe_path]
        try:
            res = subprocess.run(cmd, capture_output=True, text=True, timeout=5)
            height = int(res.stdout.strip()) if res.stdout.strip().isdigit() else 1080
        except:
            logging.warning("ffprobe not found or failed, defaulting to 1080p.")
            height = 1080
        return size_gb, height
    except Exception as e:
        logging.error(f"Metadata error: {e}")
        return 0, 1080

def remux_subtitles(video_path, sub_paths, output_path):
    try:
        cmd = ["ffmpeg", "-i", str(video_path)]
        for sub in sub_paths:
            cmd.extend(["-i", str(sub)])
        cmd.extend(["-map", "0"])
        for i in range(1, len(sub_paths) + 1):
            cmd.extend(["-map", str(i)])
        cmd.extend(["-c", "copy", "-y", str(output_path)])
        res = subprocess.run(cmd, capture_output=True, text=True)
        return res.returncode == 0
    except: return False

def generate_secure_link(source_path, remux=False):
    # En mode SYMÉTRIE, le chemin source est déjà correct
    config = load_config()
    current_base_url = config.get("base_url", BASE_URL)
    os.makedirs(SHARE_DIR, exist_ok=True)
    
    # Path translation for host environment (Radarr/Sonarr use Docker paths)
    translated_path = source_path
    if not DOCKER_MODE:
        if translated_path.startswith("/movies"):
            # Try /mnt/externe first (common in this setup)
            candidate = translated_path.replace("/movies", "/mnt/externe/movies", 1)
            if os.path.exists(candidate):
                translated_path = candidate
        elif translated_path.startswith("/tv"):
            candidate = translated_path.replace("/tv", "/mnt/externe/tv", 1)
            if os.path.exists(candidate):
                translated_path = candidate
        
        if source_path != translated_path:
            logging.info(f"Path translated: {source_path} -> {translated_path}")

    src = Path(translated_path)
    if not src.exists():
        logging.error(f"Source not found: {translated_path}")
        return None

    if src.is_dir():
        video_files = []
        for ext in ('.mkv', '.mp4', '.m2ts', '.avi', '.ts'):
            video_files.extend(list(src.rglob(f"*{ext}")))
        if not video_files:
            logging.warning(f"No video files found in directory: {translated_path}")
            return None
        src = max(video_files, key=lambda f: f.stat().st_size)
        logging.info(f"Auto-selected video file: {src.name}")

    size_gb, height = get_video_info(src)
    logging.info(f"Video info: {size_gb}GB, {height}p")
    
    if size_gb > config.get('max_gb', 4.0) or height > config.get('max_res', 1080):
        logging.warning(f"Limits exceeded: {size_gb}GB > {config.get('max_gb')}, {height}p > {config.get('max_res')}")
        return None

    timestamp = str(int(time.time()))
    unique_hash = hashlib.sha256(f"{src.name}{timestamp}".encode()).hexdigest()[:16]
    
    parent_dir = src.parent
    video_stem = src.stem
    subtitles = list(parent_dir.glob(f"{video_stem}*.srt"))
    logging.info(f"Subtitles found: {len(subtitles)}")

    if remux and subtitles:
        video_share_name = f"{unique_hash}_{src.stem}.mkv"
        video_dest_path = Path(SHARE_DIR) / video_share_name
        logging.info(f"Starting remux to {video_share_name}")
        if remux_subtitles(src, subtitles, video_dest_path):
            return {"video": f"{current_base_url}/{video_share_name}", "type": "remuxed"}
        logging.error("Remux failed")

    video_share_name = f"{unique_hash}_{src.name}"
    video_dest_path = Path(SHARE_DIR) / video_share_name
    
    try:
        if video_dest_path.exists(): video_dest_path.unlink()
        os.symlink(src, video_dest_path)
        logging.info(f"Symlink created: {video_dest_path}")
    except Exception as e:
        logging.error(f"Symlink error: {e}")
        return None

    result = {"video": f"{current_base_url}/{video_share_name}", "subtitles": [], "type": "symlink"}
    for sub in subtitles:
        sub_share_name = f"{unique_hash}_{sub.name}"
        sub_dest_path = Path(SHARE_DIR) / sub_share_name
        try:
            if sub_dest_path.exists(): sub_dest_path.unlink()
            os.symlink(sub, sub_dest_path)
            result["subtitles"].append(f"{current_base_url}/{sub_share_name}")
        except: pass
    return result
