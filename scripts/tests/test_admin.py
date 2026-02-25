import pytest
import os
import sys
import json
from unittest.mock import MagicMock, patch

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

def test_status_cmd(mock_bot, mock_is_authorized, mocker):
    import handlers_admin
    
    # Mock disk_usage
    mocker.patch('shutil.disk_usage', return_value=MagicMock(total=100*1024**3, used=50*1024**3, free=50*1024**3))
    mocker.patch('os.path.exists', return_value=True)
    
    # Trigger registration
    handlers_admin.register_admin_handlers(mock_bot, mock_is_authorized)
    
    status_func = mock_bot.registered_handlers.get('status')
    assert status_func is not None
    
    # Simulate message
    m = MagicMock()
    m.from_user.id = 123
    m.chat.id = 456
    
    status_func(m)
    
    # Check if reply_to was called with status info
    mock_bot.reply_to.assert_called_once()
    args, kwargs = mock_bot.reply_to.call_args
    assert "Statut Stockage" in args[1]
    assert "NVMe" in args[1]
    assert "HDD" in args[1]

def test_save_users(mocker):
    import handlers_admin
    mock_open = mocker.patch("builtins.open", mocker.mock_open())
    mock_json_dump = mocker.patch("json.dump")
    
    handlers_admin.save_users([1, 2, 3])
    
    mock_open.assert_called_once()
    mock_json_dump.assert_called_once()
    args, _ = mock_json_dump.call_args
    assert args[0] == {"allowed": [1, 2, 3]}

def test_admin_menu(mock_bot, mock_is_authorized):
    import handlers_admin
    handlers_admin.register_admin_handlers(mock_bot, mock_is_authorized)
    
    admin_menu_func = mock_bot.registered_handlers.get('admin')
    assert admin_menu_func is not None
    
    m = MagicMock()
    m.from_user.id = 123
    m.chat.id = 456
    
    admin_menu_func(m)
    
    mock_bot.send_message.assert_called_once()
    args, kwargs = mock_bot.send_message.call_args
    assert "Menu Administrateur" in args[1]

def test_admin_callback_users(mock_bot, mock_is_authorized):
    import handlers_admin
    handlers_admin.register_admin_handlers(mock_bot, mock_is_authorized)
    
    callback_handler = mock_bot.registered_handlers.get('callback')
    assert callback_handler is not None
    
    # Simulate clicking the "Users" button
    call = MagicMock()
    call.data = "adm:users"
    call.from_user.id = 123
    call.message.chat.id = 456
    call.message.message_id = 789
    
    callback_handler(call)
    
    # Should edit the message to show user management options
    mock_bot.edit_message_text.assert_called_once()
    args, kwargs = mock_bot.edit_message_text.call_args
    assert "Gestion des Utilisateurs" in args[0]
    # Check if buttons are present
    markup = kwargs['reply_markup']
    buttons = [b.callback_data for row in markup.keyboard for b in row]
    assert "adm:users_list" in buttons
    assert "adm:users_add" in buttons

def test_process_user_add_logic(mock_bot, mocker):
    import handlers_admin
    # Mocking USERS_FILE reading
    mocker.patch("builtins.open", mocker.mock_open(read_data='{"allowed": [123]}'))
    mocker.patch("handlers_admin.save_users")
    
    # Case 1: Valid numeric ID
    m = MagicMock()
    m.text = "456"
    handlers_admin.register_admin_handlers(mock_bot, lambda uid: True)
    
    # We need to find the process_user_add function. 
    # It's local to register_admin_handlers, but we can capture it via register_next_step_handler
    
    # Simulate triggering the callback that registers the next step
    callback_handler = mock_bot.registered_handlers.get('callback')
    call = MagicMock()
    call.data = "adm:users_add"
    callback_handler(call)
    
    # Capture the function passed to register_next_step_handler
    assert mock_bot.register_next_step_handler.called
    process_user_add = mock_bot.register_next_step_handler.call_args[0][1]
    
    # Test valid addition
    process_user_add(m)
    mock_bot.reply_to.assert_called_with(m, "✅ ID 456 ajouté.")
    
    # Case 2: Invalid non-numeric ID
    m.text = "not_an_id"
    process_user_add(m)
    assert "ID invalide" in mock_bot.reply_to.call_args[0][1]
