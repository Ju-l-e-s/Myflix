import os
import sys
from unittest.mock import MagicMock

# Add scripts to path
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

# --- media_manager.py tests ---


def test_clean_media_name():
    import media_manager

    assert (
        media_manager.clean_media_name("Inception.2010.1080p.BluRay.x264.mkv")
        == "Inception (2010)"
    )
    assert (
        media_manager.clean_media_name("The.Boys.S01E01.720p.WEB-DL.mkv") == "The Boys"
    )
    assert (
        media_manager.clean_media_name("Avatar.The.Way.Of.Water.2160p.4k.HEVC.mkv")
        == "Avatar The Way Of Water"
    )


def test_get_size_file(tmp_path):
    import media_manager

    f = tmp_path / "test.mkv"
    f.write_text("content" * 1000)
    assert media_manager.get_size(str(f)) == f.stat().st_size


def test_get_size_dir(tmp_path):
    import media_manager

    d = tmp_path / "test_dir"
    d.mkdir()
    f1 = d / "f1.mkv"
    f1.write_text("abc")
    f2 = d / "f2.mkv"
    f2.write_text("def")
    # Note: media_manager.get_size uses os.walk(p) for r, d, f in os.walk(p) for f in fs
    # Actually the code has: sum(os.path.getsize(os.path.join(r, d, f)) for r, d, fs in os.walk(p) for f in fs)
    # Wait, the code I read was: sum(os.path.getsize(os.path.join(r, d, f)) for r, d, fs in os.walk(p) for f in fs)
    # Actually it should be os.path.join(r, f). Let's check that.
    assert media_manager.get_size(str(d)) == 6


def test_find_local_media(mocker):
    import media_manager

    mocker.patch("os.path.exists", return_value=True)
    mocker.patch(
        "os.listdir", side_effect=lambda p: ["Titanic.1997.mkv"] if "Films" in p else []
    )
    mocker.patch("media_manager.get_size", return_value=2 * 1024**3)

    results = media_manager.find_local_media("titanic")
    assert len(results) == 1
    assert results[0]["clean_name"] == "Titanic (1997)"
    assert results[0]["is_nvme"] is False  # HDD path in PATHS['movies']


# --- storage_manager.py tests ---


def test_storage_manager_list_items(mocker):
    import storage_manager

    mocker.patch("os.path.exists", return_value=True)
    mocker.patch("os.listdir", return_value=["Movie_A", "Movie_B"])
    mocker.patch("storage_manager.get_size", return_value=10.5)

    out = storage_manager.list_items("/tmp/fake")
    assert "Movie_A (10.5G)" in out
    assert "/arc_Movie_A" in out


# --- storage_skill.py tests ---


def test_storage_skill_get_status(mocker):
    import storage_skill

    mock_usage = MagicMock(total=100 * 1024**3, used=40 * 1024**3, free=60 * 1024**3)
    mocker.patch("shutil.disk_usage", return_value=mock_usage)

    status = storage_skill.get_status()
    assert "NVMe : 40.0%" in status
    assert "60 Go libres" in status
