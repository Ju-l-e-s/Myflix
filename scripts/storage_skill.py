import os
import shutil
import sys

NVME_BUFFER = "/data/media/buffer"
HDD_BASE = "/mnt/externe"
PATHS = {"movies": f"{HDD_BASE}/Films", "series": f"{HDD_BASE}/Series"}


def list_media(category="movies"):
    path = PATHS.get(category)
    if not os.path.exists(path):
        return f"❌ Dossier {category} introuvable."
    items = [f for f in os.listdir(path) if not f.startswith(".")]
    return (
        f"🎥 **{category.capitalize()} :**\n"
        + "\n".join([f"• {i}" for i in sorted(items)])
        if items
        else f"📂 Aucun contenu dans {category}."
    )


def get_status():
    report = "📊 **État du Stockage :**\n"
    for name, path in [("🚀 NVMe", "/"), ("📚 HDD", HDD_BASE)]:
        usage = shutil.disk_usage(path)
        report += f"{name} : {(usage.used / usage.total) * 100:.1f}% ({usage.free // (2**30)} Go libres)\n"
    return report


if __name__ == "__main__":
    if len(sys.argv) > 1:
        cmd = sys.argv[1]
        if cmd == "list_media" and len(sys.argv) > 2:
            print(list_media(sys.argv[2]))
        elif cmd == "get_status":
            print(get_status())
