import os
import shutil
import sys

NVME_BUFFER = "/data/media/buffer"
HDD_BASE = "/mnt/externe"
PATHS = {"movies": f"{HDD_BASE}/Films", "series": f"{HDD_BASE}/Series"}


def create_status_msg(used_gb, total_gb, label, icon, tier_label):
    pct = (used_gb / total_gb) * 100
    free_gb = total_gb - used_gb

    # Seuils standards
    if pct > 90:
        status_emoji = "🔴"
    elif pct > 75:
        status_emoji = "🟡"
    else:
        status_emoji = "🟢"

    # Barre de 18 caractères sans backticks
    bar_width = 18
    filled = int((pct / 100) * bar_width)
    filled = min(filled, bar_width)
    bar = "█" * filled + "░" * (bar_width - filled)
    
    # Choix de l'icône de données
    data_icon = "📥" if icon == "🚀" else "📂"

    return (
        f"{icon} {label} ({tier_label})\n"
        f"{bar} {pct:.1f}%\n"
        f"{data_icon} {used_gb:.1f} / {total_gb:.1f} GB\n"
        f"{status_emoji} Libre : {free_gb:.1f} GB\n"
    )


def get_status():
    report = "🏛 SYSTÈME : ÉTAT DU STOCKAGE\n\n"
    for name, path, icon, tier in [
        ("NVMe", "/", "🚀", "Hot Tier"),
        ("HDD", HDD_BASE, "📚", "Archive"),
    ]:
        if not os.path.exists(path):
            continue
        usage = shutil.disk_usage(path)
        used = usage.used / (2**30)
        total = usage.total / (2**30)
        report += create_status_msg(used, total, name, icon, tier) + "\n"

    report += "🛰 Statut : Opérationnel"
    return report


if __name__ == "__main__":
    if len(sys.argv) > 1:
        cmd = sys.argv[1]
        if cmd == "get_status":
            print(get_status())
