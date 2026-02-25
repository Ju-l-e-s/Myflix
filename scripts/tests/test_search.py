import pytest
import os
import sys
import json
from unittest.mock import MagicMock

# Add scripts to path
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

@pytest.fixture
def mock_bot():
    bot = MagicMock()
    bot.registered_handlers = {}
    
    def message_handler_decorator(**kwargs):
        def decorator(f):
            # Capture handlers by identifying them (commands or func)
            if 'commands' in kwargs:
                for cmd in kwargs['commands']:
                    bot.registered_handlers[cmd] = f
            else:
                bot.registered_handlers['ai_handler'] = f
            return f
        return decorator
    
    bot.message_handler.side_effect = message_handler_decorator
    
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

def test_stream_gpt_json_success(mocker):
    import handlers_search
    mock_response = MagicMock()
    mock_response.json.return_value = {
        'choices': [{'message': {'content': '{"titre": "Titanic", "type": "film"}'}}]
    }
    mocker.patch('requests.post', return_value=mock_response)
    
    res = handlers_search.stream_gpt_json("Titanic")
    assert 'Titanic' in res
    assert 'film' in res

def test_stream_gpt_json_failure(mocker):
    import handlers_search
    mocker.patch('requests.post', side_effect=Exception("API Error"))
    res = handlers_search.stream_gpt_json("Titanic")
    assert res is None

def test_process_get_request_success(mock_bot, mocker):
    import handlers_search
    # Mock stream_gpt_json to return valid JSON
    mocker.patch('handlers_search.stream_gpt_json', return_value='''```json
{"titre": "Titanic", "type": "film"}
```''')
    
    # Mock Radarr lookup API
    mock_results = [{"title": "Titanic", "year": 1997, "titleSlug": "titanic-1997", "tmdbId": 597, "images": []}]
    mock_response = MagicMock()
    mock_response.json.return_value = mock_results
    mocker.patch('requests.get', return_value=mock_response)
    
    # Mock context file
    mocker.patch('os.path.exists', return_value=False)
    mocker.patch('builtins.open', mocker.mock_open())
    mocker.patch('json.dump')
    
    m = MagicMock()
    m.chat.id = 456
    
    handlers_search.process_get_request(mock_bot, m, "Titanic")
    
    mock_bot.send_message.assert_called_once()
    args, kwargs = mock_bot.send_message.call_args
    assert "Résultats pour 'Titanic'" in args[1]
    assert "Titanic (1997)" in args[1]
    assert kwargs['parse_mode'] == 'Markdown'

def test_process_get_request_no_results(mock_bot, mocker):
    import handlers_search
    mocker.patch('handlers_search.stream_gpt_json', return_value='{"titre": "Nonexistent", "type": "film"}')
    
    mock_response = MagicMock()
    mock_response.json.return_value = []
    mocker.patch('requests.get', return_value=mock_response)
    
    m = MagicMock()
    m.chat.id = 456
    handlers_search.process_get_request(mock_bot, m, "Nonexistent")
    
    mock_bot.send_message.assert_called_with(456, "❌ Aucun résultat.")

def test_download_callback_handler_success(mock_bot, mock_is_authorized, mocker):
    import handlers_search
    handlers_search.register_search_handlers(mock_bot, mock_is_authorized)
    
    callback_handler = mock_bot.registered_handlers.get('callback')
    assert callback_handler is not None
    
    # Mock context data
    mock_ctx = {
        "456_search": {
            "cat": "movie",
            "results": [{"title": "Titanic", "year": 1997, "titleSlug": "titanic-1997", "tmdbId": 597, "images": [], "id": 0}]
        }
    }
    mocker.patch('builtins.open', mocker.mock_open(read_data=json.dumps(mock_ctx)))
    
    # Mock Radarr add API
    mock_response = MagicMock()
    mock_response.status_code = 201
    mocker.patch('requests.post', return_value=mock_response)
    
    call = MagicMock()
    call.data = "dl_sel:movie:0"
    call.from_user.id = 123
    call.message.chat.id = 456
    call.message.message_id = 789
    
    callback_handler(call)
    
    mock_bot.edit_message_text.assert_called_once()
    assert "Ajouté :** `Titanic (1997)`" in mock_bot.edit_message_text.call_args[0][0]

def test_download_callback_session_expired(mock_bot, mock_is_authorized, mocker):
    import handlers_search
    handlers_search.register_search_handlers(mock_bot, mock_is_authorized)
    callback_handler = mock_bot.registered_handlers.get('callback')
    
    # Mock empty context
    mocker.patch('builtins.open', mocker.mock_open(read_data='{}'))
    
    call = MagicMock()
    call.data = "dl_sel:movie:0"
    call.message.chat.id = 456
    
    callback_handler(call)
    
    mock_bot.answer_callback_query.assert_called_with(call.id, "❌ Session expirée.")
