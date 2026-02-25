import pytest
import os
import sys
from unittest.mock import MagicMock

# Add scripts to path
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))


@pytest.fixture
def mock_paths(mocker):
    # Patch global variables in cleanup_share
    mocker.patch("cleanup_share.SHARE_DIR", "/tmp/share")
    mocker.patch("cleanup_share.NVME_PATH", "/tmp/nvme")
    mocker.patch("cleanup_share.HDD_PATH", "/tmp/hdd")
    mocker.patch("cleanup_share.LOG_FILE", "/tmp/cleanup.log")
    # Prevent actual logging to file
    mocker.patch("logging.basicConfig")
    mocker.patch("logging.info")
    mocker.patch("logging.error")
    mocker.patch("logging.warning")


def test_purge_share_directory_success(mock_paths, mocker):
    import cleanup_share

    mocker.patch("os.path.exists", return_value=True)

    # Mock os.walk to return one file and one directory
    # root, dirs, files
    mocker.patch(
        "os.walk",
        return_value=[
            ("/tmp/share", ["subfolder"], ["old_file.mkv"]),
            ("/tmp/share/subfolder", [], []),
        ],
    )

    # Mock lstat to return an old mtime for the file and a new one for the folder
    mock_stat_old = MagicMock()
    mock_stat_old.st_mtime = 0

    mock_stat_new = MagicMock()
    mock_stat_new.st_mtime = 200000

    def side_effect_lstat(path):
        if "old_file.mkv" in path:
            return mock_stat_old
        return mock_stat_new

    mocker.patch("os.lstat", side_effect=side_effect_lstat)
    mocker.patch("time.time", return_value=100000)

    mock_unlink = mocker.patch("os.unlink")
    mock_rmtree = mocker.patch("shutil.rmtree")

    # Mock type checks
    mocker.patch("os.path.islink", side_effect=lambda p: "old_file.mkv" in p)
    mocker.patch("os.path.isfile", return_value=False)
    mocker.patch("os.path.isdir", side_effect=lambda p: "subfolder" in p)

    cleanup_share.purge_share_directory()

    # Should unlink the old file but NOT rmtree the new folder
    mock_unlink.assert_called_once_with("/tmp/share/old_file.mkv")
    mock_rmtree.assert_not_called()


def test_auto_tiering_nvme_to_hdd_not_needed(mock_paths, mocker):
    import cleanup_share

    # 20% used, target is 50%
    mocker.patch(
        "shutil.disk_usage", return_value=MagicMock(total=100, used=20, free=80)
    )
    mocker.patch("cleanup_share.is_hdd_mounted", return_value=True)

    mock_walk = mocker.patch("os.walk")

    cleanup_share.auto_tiering_nvme_to_hdd()

    mock_walk.assert_not_called()


def test_auto_tiering_hdd_not_mounted(mock_paths, mocker):
    import cleanup_share

    mocker.patch("cleanup_share.is_hdd_mounted", return_value=False)
    mock_usage = mocker.patch("cleanup_share.get_nvme_usage_percent")

    cleanup_share.auto_tiering_nvme_to_hdd()

    mock_usage.assert_not_called()


def test_auto_tiering_migration_flow(mock_paths, mocker):
    import cleanup_share

    # 80% used, target is 50%
    # First call: check if needed
    # Second call: check if still needed during loop (before move)
    # Third call: check after move (now it's 40%)
    mocker.patch(
        "shutil.disk_usage",
        side_effect=[
            MagicMock(total=100, used=80, free=20),
            MagicMock(total=100, used=80, free=20),
            MagicMock(total=100, used=40, free=60),
        ],
    )
    mocker.patch("cleanup_share.is_hdd_mounted", return_value=True)

    # Mock os.walk to find a movie
    mocker.patch(
        "os.walk", return_value=[("/tmp/nvme/Movies/Titanic", [], ["titanic.mkv"])]
    )
    mocker.patch("os.path.islink", return_value=False)
    # st_atime=100 (old), size=40
    mocker.patch("os.stat", return_value=MagicMock(st_atime=100, st_size=40))

    mock_makedirs = mocker.patch("os.makedirs")
    mock_move = mocker.patch("shutil.move")
    mock_symlink = mocker.patch("os.symlink")

    cleanup_share.auto_tiering_nvme_to_hdd()

    # Path translation: rel_path should be 'Movies/Titanic/titanic.mkv'
    # dest_path should be '/tmp/hdd/Movies/Titanic/titanic.mkv'
    mock_move.assert_called_once_with(
        "/tmp/nvme/Movies/Titanic/titanic.mkv", "/tmp/hdd/Movies/Titanic/titanic.mkv"
    )
    mock_symlink.assert_called_once_with(
        "/tmp/hdd/Movies/Titanic/titanic.mkv", "/tmp/nvme/Movies/Titanic/titanic.mkv"
    )
    mock_makedirs.assert_called_once_with("/tmp/hdd/Movies/Titanic", exist_ok=True)


def test_auto_tiering_rollback_on_failure(mock_paths, mocker):
    import cleanup_share

    mocker.patch(
        "shutil.disk_usage", return_value=MagicMock(total=100, used=80, free=20)
    )
    mocker.patch("cleanup_share.is_hdd_mounted", return_value=True)
    mocker.patch("os.walk", return_value=[("/tmp/nvme/Movies", [], ["fail.mkv"])])
    mocker.patch("os.path.islink", return_value=False)
    mocker.patch("os.stat", return_value=MagicMock(st_atime=100, st_size=10))

    mocker.patch("os.makedirs")
    mocker.patch("shutil.move")
    # Symlink fails!
    mocker.patch("os.symlink", side_effect=Exception("Symlink failed"))

    # Rollback mocks
    mocker.patch(
        "os.path.exists", side_effect=lambda p: p == "/tmp/hdd/Movies/fail.mkv"
    )
    mock_rollback_move = mocker.patch(
        "shutil.move"
    )  # Will be called again for rollback

    cleanup_share.auto_tiering_nvme_to_hdd()

    # shulil.move should have been called twice (one for move, one for rollback)
    assert mock_rollback_move.call_count == 2


def test_ensure_hdd_symlinks_creates_missing_link(mock_paths, mocker):
    import cleanup_share

    # Mock existence des dossiers HDD
    mocker.patch(
        "os.path.exists",
        side_effect=lambda p: p in ["/tmp/hdd/Films", "/tmp/hdd/Series"],
    )
    mocker.patch("os.makedirs")

    # Mock contenu du HDD (un film présent)
    mocker.patch(
        "os.listdir", side_effect=lambda p: ["Zootopia_2_2025"] if "Films" in p else []
    )

    # Simuler que le lien NVMe n'existe PAS
    # os.path.exists('/tmp/nvme/movies/Zootopia_2_2025') doit retourner False
    mocker.patch("os.path.exists", side_effect=lambda p: False if "nvme" in p else True)

    mock_symlink = mocker.patch("os.symlink")

    cleanup_share.ensure_hdd_symlinks()

    # Vérifier que symlink a été appelé avec les bons chemins
    # Note: mapping utilise os.path.join(NVME_PATH, "movies") -> /tmp/nvme/movies
    mock_symlink.assert_called_once_with(
        "/tmp/hdd/Films/Zootopia_2_2025", "/tmp/nvme/movies/Zootopia_2_2025"
    )
