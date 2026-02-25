import os
import sys

# Add scripts to path so we can import config
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

def test_config_vars():
    import config
    import re
    assert hasattr(config, "TOKEN")
    # Telegram token format: bot_id:alphanumeric_string (approx 45-50 chars total)
    token_pattern = r"^\d+:[a-zA-Z0-9_-]{35,}$"
    assert re.match(token_pattern, config.TOKEN), f"Token format invalid: {config.TOKEN}"
    
    assert hasattr(config, "SUPER_ADMIN")
    assert hasattr(config, "USERS_FILE")
    assert hasattr(config, "RADARR_CFG")
    assert hasattr(config, "SONARR_CFG")
    assert hasattr(config, "QBIT_URL")
    assert hasattr(config, "OPENAI_KEY")

def test_config_urls():
    import config
    assert config.RADARR_CFG["url"].startswith("http")
    assert config.SONARR_CFG["url"].startswith("http")
    assert config.QBIT_URL.startswith("http")

def test_is_authorized_super_admin():
    import config
    assert config.is_authorized(config.SUPER_ADMIN) == True
