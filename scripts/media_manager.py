import os
import shutil
import sys
import json
import re

DOCKER_MODE = os.environ.get("DOCKER_MODE") == "true"
HOME = os.path.expanduser("~")

# Avec MergerFS, on pointe sur le pool unifié pour la lecture
POOL_MEDIA = "/data/media" if DOCKER_MODE else "/mnt/pool/media"

PATHS = {
    "movies": os.path.join(POOL_MEDIA, "movies"),
    "series": os.path.join(POOL_MEDIA, "tv"),
}

# Pour les actions d'archivage (écriture physique)
NVME_ROOT = "/home/jules/data"
HDD_ROOT = "/mnt/externe"

MIN_SIZE_BYTES = 500 * 1024 * 1024

def clean_title(title):
    pattern = r'(?i)(?:1080p|720p|4k|uhd|x26[45]|h26[45]|web[- ]?(dl|rip)|bluray|aac|dd[p]?5\.1|atmos|repack|playweb|max|[\d\s]+A M|[\d\s]+P M|-NTb|-playWEB)'
    parts = re.split(pattern, title)
    cleaned = parts[0].replace('.', ' ').strip()
    return re.sub(r'\s+', ' ', cleaned)

def clean_media_name(name):
    ep_match = re.search(r"S(\d+)E(\d+)", name, re.IGNORECASE)
    ep_info = ep_match.group(0).upper() if ep_match else ""
    title = clean_title(name)
    if ep_info:
        title = title.replace(ep_info, "").replace(ep_info.lower(), "").strip()
        return f"[{ep_info}] {title}"
    return title

def get_progress_bar(percentage, length=15):
    """Génère une barre de progression élégante."""
    filled_length = int(length * percentage / 100)
    bar = "█" * filled_length + "░" * (length - filled_length)
    return f"`{bar}` {percentage}%"

def format_size(size_bytes):
    """Formate la taille en GB ou MB."""
    if size_bytes >= 1024**3:
        return f"{round(size_bytes / 1024**3, 2)} GB"
    return f"{round(size_bytes / 1024**2, 2)} MB"

def format_speed(speed_bytes_per_sec):
    """Formate la vitesse de téléchargement."""
    if speed_bytes_per_sec >= 1024**2:
        return f"{round(speed_bytes_per_sec / 1024**2, 1)} MB/s"
    return f"{round(speed_bytes_per_sec / 1024, 1)} KB/s"

def get_size(p):
    """Calcule la taille de manière optimisée via scandir."""
    try:
        if os.path.isfile(p):
            return os.path.getsize(p)
        total = 0
        with os.scandir(p) as it:
            for entry in it:
                if entry.is_file():
                    total += entry.stat().st_size
                elif entry.is_dir():
                    total += get_size(entry.path)
        return total
    except Exception: return 0

def find_local_media(query):
    results = []
    q = query.lower().replace(" ", "")
    for cat, d in PATHS.items():
        if not os.path.exists(d): continue
        for item in os.listdir(d):
            if q in item.lower().replace(" ", "").replace(".", ""):
                path = os.path.join(d, item)
                results.append({
                    "name": item,
                    "clean_name": clean_media_name(item),
                    "path": path,
                    "size_gb": round(get_size(path) / (1024**3), 2),
                    "category": cat,
                })
    return results

def get_media_list(category):
    media = []
    root_dir = PATHS.get(category)
    if not root_dir or not os.path.exists(root_dir): return []
    for f in os.listdir(root_dir):
        if f.startswith("."): continue
        path = os.path.join(root_dir, f)
        size = get_size(path)
        if size < MIN_SIZE_BYTES: continue
        media.append({
            "name": clean_media_name(f),
            "real_name": f,
            "size_gb": round(size / (1024**3), 2),
            "path": path,
        })
    return sorted(media, key=lambda x: x["size_gb"], reverse=True)

if __name__ == "__main__":
    if len(sys.argv) < 2: sys.exit(1)
    cmd = sys.argv[1]
    if cmd == "json_list":
        print(json.dumps(get_media_list(sys.argv[2])))
    elif cmd == "search_local":
        print(json.dumps(find_local_media(sys.argv[2])))
    elif cmd == "status":
        u = shutil.disk_usage(NVME_ROOT)
        h = shutil.disk_usage(HDD_ROOT) if os.path.exists(HDD_ROOT) else u
        print(json.dumps({"nvme_free": u.free // 1024**3, "hdd_free": h.free // 1024**3}))
    elif cmd == "action":
        act, cat, path = sys.argv[2], sys.argv[3], sys.argv[4]
        if act == "del":
            if os.path.isdir(path): shutil.rmtree(path)
            else: os.remove(path)
        # L'archivage déplace physiquement du NVMe vers le HDD
        elif act == "arc":
            rel_path = os.path.relpath(path, PATHS[cat])
            dest = os.path.join(HDD_ROOT, "media", cat, rel_path) # Adaptation chemin physique
            os.makedirs(os.path.dirname(dest), exist_ok=True)
            shutil.move(path, dest)
        print("Done")
