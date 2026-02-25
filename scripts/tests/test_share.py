import pytest
import os
import sys
import json
import shutil
from pathlib import Path
from unittest.mock import MagicMock, patch

# Add scripts to path
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

@pytest.fixture
def mock_config_file(tmp_path):
    config_data = {"max_gb": 100.0, "max_res": 4320, "base_url": "https://test.share.fr"}
    config_file = tmp_path / "config_share.json"
    with open(config_file, "w") as f:
        json.dump(config_data, f)
    return config_file

@pytest.fixture
def share_dir(tmp_path):
    d = tmp_path / "share"
    d.mkdir()
    return d

def test_get_video_info_real_file(tmp_path, mocker):
    import share_engine
    test_file = tmp_path / "movie.mkv"
    # 2GB file
    with open(test_file, "wb") as f:
        f.truncate(2 * 1024**3)
    
    # Mock ffprobe output
    mock_res = MagicMock()
    mock_res.stdout = "1080\n"
    mocker.patch('subprocess.run', return_value=mock_res)
    
    size, res = share_engine.get_video_info(test_file)
    assert size == 2.0
    assert res == 1080

def test_generate_secure_link_success(mock_config_file, share_dir, tmp_path, mocker):
    import share_engine
    # Setup paths
    mocker.patch('share_engine.CONFIG_FILE', str(mock_config_file))
    mocker.patch('share_engine.SHARE_DIR', str(share_dir))
    mocker.patch('share_engine.DOCKER_MODE', True) # Set to True to skip path translation logic for simple test
    
    # Create source
    src = tmp_path / "movie.mkv"
    src.write_text("content")
    
    # Mock video info to pass limits
    mocker.patch('share_engine.get_video_info', return_value=(1.0, 720))
    
    result = share_engine.generate_secure_link(str(src))
    
    assert result is not None
    assert "https://test.share.fr" in result['video']
    assert result['type'] == 'symlink'

def test_generate_secure_link_limit_exceeded(mock_config_file, share_dir, tmp_path, mocker):
    import share_engine
    mocker.patch('share_engine.CONFIG_FILE', str(mock_config_file))
    mocker.patch('share_engine.SHARE_DIR', str(share_dir))
    
    src = tmp_path / "huge_movie.mkv"
    src.write_text("content")
    
    # Mock video info to exceed limits (150GB > 100GB)
    mocker.patch('share_engine.get_video_info', return_value=(150.0, 1080))
    
    result = share_engine.generate_secure_link(str(src))
    assert result is None
