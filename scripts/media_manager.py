import os, shutil, sys, json, re, subprocess

DOCKER_MODE = os.environ.get("DOCKER_MODE") == "true"
NVME_MOVIES = "/data/internal/media/movies" if DOCKER_MODE else "/home/jules/infra/ai/data/media/movies"
NVME_TV = "/data/internal/media/tv" if DOCKER_MODE else "/home/jules/infra/ai/data/media/tv"
HDD_ROOT = "/data/external" if DOCKER_MODE else "/mnt/externe"

PATHS = {
    "movies": {"nvme": [NVME_MOVIES], "hdd": f"{HDD_ROOT}/Films"},
    "series": {"nvme": [NVME_TV], "hdd": f"{HDD_ROOT}/Series"}
}
MIN_SIZE_BYTES = 500 * 1024 * 1024

def clean_media_name(name):
    name = os.path.splitext(name)[0]
    name = re.sub(r'[\._]', ' ', name)
    match = re.search(r'(.*?)\s*[\(\[]?((?:19|20)\d{2})[\)\]]?(.*)', name, re.IGNORECASE)
    title = match.group(1).strip() if match else name
    year = match.group(2) if match else ""
    tags = [r'\b\d{3,4}p\b', r'\b2160p\b', r'\b4k\b', r'\bHEVC\b', r'\bx26[45]\b', r'\bBluRay\b', r'\bWEB-DL\b', r'\bCOMPLETE\b', r'\bSeason\s*\d*\b', r'\bS\d+E\d+\b']
    clean = f"{title} ({year})" if year else title
    for tag in tags: clean = re.sub(tag, '', clean, flags=re.IGNORECASE)
    return re.sub(r'\s+', ' ', clean).strip()

def get_size(p):
    try:
        if os.path.isfile(p): return os.path.getsize(p)
        return sum(os.path.getsize(os.path.join(r, f)) for r, d, fs in os.walk(p) for f in fs)
    except Exception as e:
        return 0

def find_local_media(query):
    """Cherche un média sur les disques et renvoie les infos techniques."""
    results = []
    q = query.lower().replace(" ", "")
    for cat in PATHS:
        for d, is_nvme in [(p, True) for p in PATHS[cat]["nvme"]] + [(PATHS[cat]["hdd"], False)]:
            if not os.path.exists(d): continue
            for item in os.listdir(d):
                if q in item.lower().replace(" ", "").replace(".", ""):
                    path = os.path.join(d, item)
                    results.append({
                        "name": item,
                        "clean_name": clean_media_name(item),
                        "path": path,
                        "is_nvme": is_nvme,
                        "size_gb": round(get_size(path) / (1024**3), 2),
                        "category": cat
                    })
    return results

def get_media_list(category):
    media = []
    conf = PATHS.get(category)
    if not conf: return []
    for d, is_nvme in [(p, True) for p in conf["nvme"]] + [(conf["hdd"], False)]:
        if not os.path.exists(d): continue
        for f in os.listdir(d):
            if f.startswith('.'): continue
            path = os.path.join(d, f); size = get_size(path)
            if size < MIN_SIZE_BYTES: continue
            media.append({"name": clean_media_name(f), "real_name": f, "size_gb": round(size / (1024**3), 2), "is_nvme": is_nvme, "path": path})
    return sorted(media, key=lambda x: x['size_gb'], reverse=True)

if __name__ == "__main__":
    if len(sys.argv) < 2: sys.exit(1)
    cmd = sys.argv[1]
    if cmd == "json_list": print(json.dumps(get_media_list(sys.argv[2])))
    elif cmd == "search_local": print(json.dumps(find_local_media(sys.argv[2])))
    elif cmd == "status":
        u = shutil.disk_usage("/"); h = shutil.disk_usage(HDD_ROOT) if os.path.exists(HDD_ROOT) else u
        print(json.dumps({"nvme_free": u.free//1024**3, "hdd_free": h.free//1024**3}))
    elif cmd == "action":
        act, cat, path = sys.argv[2], sys.argv[3], sys.argv[4]
        if act == "del":
            if os.path.isdir(path): shutil.rmtree(path)
            else: os.remove(path)
        elif act == "arc": shutil.move(path, os.path.join(PATHS[cat]["hdd"], os.path.basename(path)))
        elif act == "res": shutil.move(path, os.path.join(PATHS[cat]["nvme"][0], os.path.basename(path)))
        print("Done")
