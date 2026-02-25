import os
import sys

NVME_DIR = "/data/media/films"
HDD_DIR = "/mnt/externe/Films"

def get_size(p):
    s = 0
    try:
        if os.path.isfile(p): s = os.path.getsize(p)
        else:
            for r, d, f in os.walk(p):
                for file in f: s += os.path.getsize(os.path.join(r, file))
    except: return 0
    return round(s / (1024**3), 2)

def list_items(path):
    if not os.path.exists(path): return f"❌ {path} absent."
    items = [f for f in os.listdir(path) if not f.startswith('.')]
    out = f"📂 **{os.path.basename(path)}** :\n"
    for i in sorted(items):
        sz = get_size(os.path.join(path, i))
        slug = i.replace(" ", "_")[:20]
        out += f"• {i} ({sz}G)\n  [/arc_{slug}] [/del_{slug}]\n"
    return out

if __name__ == "__main__":
    arg = sys.argv[1] if len(sys.argv) > 1 else ""
    if arg == "nvme": print(list_items(NVME_DIR))
    elif arg == "hdd": print(list_items(HDD_DIR))
    else: print("Commande reçue : " + arg)
