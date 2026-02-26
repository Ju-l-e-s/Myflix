import os
import shutil
import sys
import json
import re

DOCKER_MODE = os.environ.get("DOCKER_MODE") == "true"
HOME = os.path.expanduser("~")
NVME_MOVIES = (
    "/data/internal/media/movies"
    if DOCKER_MODE
    else os.path.join(HOME, "infra/ai/data/media/movies")
)
NVME_TV = (
    "/data/internal/media/tv"
    if DOCKER_MODE
    else os.path.join(HOME, "infra/ai/data/media/tv")
)
HDD_ROOT = "/data/external" if DOCKER_MODE else "/mnt/externe"

PATHS = {
    "movies": {"nvme": [NVME_MOVIES], "hdd": f"{HDD_ROOT}/Films"},
    "series": {"nvme": [NVME_TV], "hdd": f"{HDD_ROOT}/Series"},
}
MIN_SIZE_BYTES = 500 * 1024 * 1024


def clean_title(title):
    """Supprime les tags techniques et pollutions visuelles [cite: 2025-12-05]."""
    pattern = r'(?i)(?:1080p|720p|4k|uhd|x26[45]|h26[45]|web[- ]?(dl|rip)|bluray|aac|dd[p]?5\.1|atmos|repack|playweb|max|[\d\s]+A M|[\d\s]+P M|-NTb|-playWEB)'
    # On coupe au premier tag technique
    parts = re.split(pattern, title)
    cleaned = parts[0]
    # Nettoyage final : points remplacés par espaces, suppression doubles espaces
    cleaned = cleaned.replace('.', ' ').strip()
    return re.sub(r'\s+', ' ', cleaned)


def clean_media_name(name):
    """Version évoluée pour la queue : [S01E01] Titre propre."""
    ep_match = re.search(r"S(\d+)E(\d+)", name, re.IGNORECASE)
    ep_info = ep_match.group(0).upper() if ep_match else ""
    
    title = clean_title(name)
    
    # On retire l'épisode du titre s'il y est resté
    if ep_info:
        title = title.replace(ep_info, "").replace(ep_info.lower(), "").strip()
        return f"[{ep_info}] {title}"
    return title


def get_progress_bar(percentage, length=15):
    """Génère une barre de progression élégante [cite: 2025-12-05]."""
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
    try:
        if os.path.isfile(p):
            return os.path.getsize(p)
        return sum(
            os.path.getsize(os.path.join(r, f)) for r, d, fs in os.walk(p) for f in fs
        )
    except Exception:
        return 0


def find_local_media(query):
    """Cherche un média sur les disques et renvoie les infos techniques."""
    results = []
    q = query.lower().replace(" ", "")
    for cat in PATHS:
        for d, is_nvme in [(p, True) for p in PATHS[cat]["nvme"]] + [
            (PATHS[cat]["hdd"], False)
        ]:
            if not os.path.exists(d):
                continue
            for item in os.listdir(d):
                if q in item.lower().replace(" ", "").replace(".", ""):
                    path = os.path.join(d, item)
                    results.append(
                        {
                            "name": item,
                            "clean_name": clean_media_name(item),
                            "path": path,
                            "is_nvme": is_nvme,
                            "size_gb": round(get_size(path) / (1024**3), 2),
                            "category": cat,
                        }
                    )
    return results


def get_media_list(category):
    media = []
    conf = PATHS.get(category)
    if not conf:
        return []
    for d, is_nvme in [(p, True) for p in conf["nvme"]] + [(conf["hdd"], False)]:
        if not os.path.exists(d):
            continue
        for f in os.listdir(d):
            if f.startswith("."):
                continue
            path = os.path.join(d, f)
            size = get_size(path)
            if size < MIN_SIZE_BYTES:
                continue
            media.append(
                {
                    "name": clean_media_name(f),
                    "real_name": f,
                    "size_gb": round(size / (1024**3), 2),
                    "is_nvme": is_nvme,
                    "path": path,
                }
            )
    return sorted(media, key=lambda x: x["size_gb"], reverse=True)


if __name__ == "__main__":
    if len(sys.argv) < 2:
        sys.exit(1)
    cmd = sys.argv[1]
    if cmd == "json_list":
        print(json.dumps(get_media_list(sys.argv[2])))
    elif cmd == "search_local":
        print(json.dumps(find_local_media(sys.argv[2])))
    elif cmd == "status":
        u = shutil.disk_usage("/")
        h = shutil.disk_usage(HDD_ROOT) if os.path.exists(HDD_ROOT) else u
        print(
            json.dumps({"nvme_free": u.free // 1024**3, "hdd_free": h.free // 1024**3})
        )
    elif cmd == "action":
        act, cat, path = sys.argv[2], sys.argv[3], sys.argv[4]
        if act == "del":
            if os.path.isdir(path):
                shutil.rmtree(path)
            else:
                os.remove(path)
        elif act == "arc":
            shutil.move(path, os.path.join(PATHS[cat]["hdd"], os.path.basename(path)))
        elif act == "res":
            shutil.move(
                path, os.path.join(PATHS[cat]["nvme"][0], os.path.basename(path))
            )
        print("Done")
