import pytest
import os
import sys
from unittest.mock import MagicMock

# Add scripts to path
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

@pytest.fixture
def mock_bot():
    bot = MagicMock()
    bot.registered_handlers = {}
    
    def message_handler_decorator(**kwargs):
        def decorator(f):
            for cmd in kwargs.get('commands', []):
                bot.registered_handlers[cmd] = f
            return f
        return decorator
    
    bot.message_handler.side_effect = message_handler_decorator
    
    # Also handle callback query handlers
    def callback_query_decorator(**kwargs):
        def decorator(f):
            bot.registered_handlers['callback'] = f
            return f
        return decorator
    bot.callback_query_handler.side_effect = callback_query_decorator
    
    return bot

@pytest.fixture
def mock_is_authorized():
    return lambda uid: True

def test_show_media_list_radarr(mock_bot, mocker):
    import handlers_media
    
    # Mock requests.get
    mock_data = [
        {"id": 1, "title": "Movie 1", "year": 2021, "added": "2021-01-01T12:00:00Z"},
        {"id": 2, "title": "Movie 2", "year": 2022, "added": "2022-01-01T12:00:00Z"}
    ]
    mock_response = MagicMock()
    mock_response.json.return_value = mock_data
    mock_response.raise_for_status = MagicMock()
    mocker.patch('requests.get', return_value=mock_response)
    
    # Call show_media_list directly
    handlers_media.show_media_list(mock_bot, 456, None, "films", 0, True)
    
    # Check bot call
    mock_bot.send_message.assert_called_once()
    args, kwargs = mock_bot.send_message.call_args
    assert "Films (2)" in args[1]
    assert "Movie 2 (2022)" in args[1] # Sorted by added date reverse
    assert "Movie 1 (2021)" in args[1]

def test_show_media_list_api_error(mock_bot, mocker):
    import handlers_media
    from requests.exceptions import ConnectionError
    
    # Case 1: 401 Unauthorized
    mock_response = MagicMock()
    mock_response.raise_for_status.side_effect = Exception("401 Unauthorized")
    mocker.patch('requests.get', return_value=mock_response)
    
    handlers_media.show_media_list(mock_bot, 456, None, "films", 0, True)
    assert "Erreur API : 401 Unauthorized" in mock_bot.send_message.call_args[0][1]

    # Case 2: Service Offline (ConnectionError)
    mock_bot.send_message.reset_mock()
    mocker.patch('requests.get', side_effect=ConnectionError("Failed to connect"))
    
    handlers_media.show_media_list(mock_bot, 456, None, "films", 0, True)
    assert "Erreur API : Failed to connect" in mock_bot.send_message.call_args[0][1]

def test_show_media_list_pagination(mock_bot, mocker):
    import handlers_media
    
    # Mock data with 25 items
    mock_data = [{"id": i, "title": f"Movie {i}", "added": f"2023-01-{i:02d}T12:00:00Z"} for i in range(1, 26)]
    # Items 1-10 (page 0), 11-20 (page 1), 21-25 (page 2)
    # Sorted reverse added -> Movie 25, 24, ..., 1
    
    mock_response = MagicMock()
    mock_response.json.return_value = mock_data
    mock_response.raise_for_status = MagicMock()
    mocker.patch('requests.get', return_value=mock_response)
    
    # Test Page 1 (offset starts at Movie 15 down to Movie 6)
    # Wait, the list is: 25, 24, 23, 22, 21, 20, 19, 18, 17, 16 (Page 0)
    # 15, 14, 13, 12, 11, 10, 9, 8, 7, 6 (Page 1)
    handlers_media.show_media_list(mock_bot, 456, None, "films", 1, True)
    
    text = mock_bot.send_message.call_args[0][1]
    assert "Films (25)" in text
    assert "11. Movie 15" in text
    assert "20. Movie 6" in text
    assert "Movie 25" not in text # Should be on page 0

def test_get_storage_emoji():
    import handlers_media
    # Case 1: External HDD
    item_hdd = {'path': '/mnt/externe/media/movie.mkv', 'hasFile': True}
    assert handlers_media.get_storage_emoji(item_hdd) == "📚"
    
    # Case 2: NVMe (Path agnostique)
    item_nvme = {'path': 'media/movie.mkv', 'hasFile': True}
    assert handlers_media.get_storage_emoji(item_nvme) == "🚀"
    
    # Case 3: Missing
    item_missing = {'hasFile': False}
    assert handlers_media.get_storage_emoji(item_missing) == "🔴"

def test_queue_command(mock_bot, mock_is_authorized, mocker):
    import handlers_media
    
    # Mock qBit API
    mock_qbit_data = [
        {
            "name": "Cool.Movie.2023.1080p.BluRay.x264-GRP",
            "progress": 0.5,
            "size": 10 * 1024**3,
            "eta": 3600,
            "state": "downloading"
        }
    ]
    mock_response = MagicMock()
    mock_response.json.return_value = mock_qbit_data
    mocker.patch('requests.get', return_value=mock_response)
    
    # Call queue_command handler
    handlers_media.register_media_handlers(mock_bot, mock_is_authorized)
    
    queue_handler = mock_bot.registered_handlers.get('queue')
    assert queue_handler is not None
    
    m = MagicMock()
    m.chat.id = 456
    queue_handler(m)
    
    mock_bot.reply_to.assert_called_once()
    args, kwargs = mock_bot.reply_to.call_args
    assert "Queue" in args[1]
    assert "Cool Movie (2023)" in args[1]
    assert "50.0%" in args[1]

def test_media_callback_pagination(mock_bot, mock_is_authorized, mocker):
    import handlers_media
    handlers_media.register_media_handlers(mock_bot, mock_is_authorized)
    
    # Mock API with pagination data
    mock_data = [{"id": i, "title": f"Movie {i}", "added": f"2023-01-{i:02d}T12:00:00Z"} for i in range(1, 26)]
    mock_response = MagicMock()
    mock_response.json.return_value = mock_data
    mock_response.raise_for_status = MagicMock()
    mocker.patch('requests.get', return_value=mock_response)
    
    callback_handler = mock_bot.registered_handlers.get('callback')
    assert callback_handler is not None
    
    # Simulate clicking the "Next" (page 1) button: m_pag:films:1
    call = MagicMock()
    call.data = "m_pag:films:1"
    call.from_user.id = 123
    call.message.chat.id = 456
    call.message.message_id = 789
    
    callback_handler(call)
    
    # Should edit the message with the second page
    mock_bot.edit_message_text.assert_called_once()
    text = mock_bot.edit_message_text.call_args[0][0]
    assert "Films (25)" in text
    assert "11. Movie 15" in text # Offset for page 1

def test_queue_command_failure(mock_bot, mock_is_authorized, mocker):
    import handlers_media
    # Mock qBit API to fail
    mocker.patch('requests.get', side_effect=Exception("qBit Connection Failed"))
    
    handlers_media.register_media_handlers(mock_bot, mock_is_authorized)
    queue_handler = mock_bot.registered_handlers.get('queue')
    
    m = MagicMock()
    m.chat.id = 456
    queue_handler(m)
    
    # Should reply with "Erreur qBit"
    mock_bot.reply_to.assert_called_with(m, "❌ Erreur qBit.")
