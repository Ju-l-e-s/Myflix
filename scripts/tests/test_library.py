import pytest
import os
import sys
from unittest.mock import MagicMock

# Add scripts to path
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

@pytest.fixture
def mock_bazarr_api(mocker):
    # Mock requests.get and requests.post
    mock_get = mocker.patch('requests.get')
    mock_post = mocker.patch('requests.post')
    # Prevent logging to file
    mocker.patch('logging.basicConfig')
    mocker.patch('logging.info')
    mocker.patch('logging.error')
    mocker.patch('logging.warning')
    return mock_get, mock_post

def test_run_task_success(mock_bazarr_api):
    import maintain_library
    _, mock_post = mock_bazarr_api
    mock_response = MagicMock()
    mock_response.status_code = 200
    mock_post.return_value = mock_response
    
    assert maintain_library.run_task("update_library") is True
    mock_post.assert_called_once()
    assert "update_library" in mock_post.call_args[1]['params']['name']

def test_run_task_failure(mock_bazarr_api):
    import maintain_library
    _, mock_post = mock_bazarr_api
    mock_post.side_effect = Exception("Bazarr Offline")
    
    assert maintain_library.run_task("update_library") is False

def test_wait_for_tasks_loop(mock_bazarr_api, mocker):
    import maintain_library
    mock_get, _ = mock_bazarr_api
    
    # First call: task running, Second call: no tasks
    res1 = MagicMock()
    res1.json.return_value = [{'name': 'update_library', 'status': 'running'}]
    res2 = MagicMock()
    res2.json.return_value = [{'name': 'update_library', 'status': 'completed'}]
    
    mock_get.side_effect = [res1, res2]
    mocker.patch('time.sleep') # Skip sleep
    
    maintain_library.wait_for_tasks()
    assert mock_get.call_count == 2

def test_get_integrity_report_movies_and_series(mock_bazarr_api):
    import maintain_library
    mock_get, _ = mock_bazarr_api
    
    # 1. /api/movies
    movies_res = MagicMock()
    movies_res.json.return_value = [
        {'title': 'Movie 1', 'year': 2021, 'subtitles': [{'language': 'fr'}]},
        {'title': 'Movie 2', 'year': 2022, 'subtitles': [{'language': 'en'}]} # Missing FR
    ]
    
    # 2. /api/series
    series_res = MagicMock()
    series_res.json.return_value = [{'title': 'Series 1', 'sonarrId': 123}]
    
    # 3. /api/episodes?seriesid=123
    episodes_res = MagicMock()
    episodes_res.json.return_value = [
        {'seasonNumber': 1, 'episodeNumber': 1, 'subtitles': [{'language': 'en'}]} # Missing FR
    ]
    
    mock_get.side_effect = [movies_res, series_res, episodes_res]
    
    report = maintain_library.get_integrity_report()
    assert len(report) == 2
    assert "[Movie] Missing FR: Movie 2" in report[0]
    assert "[Series] Missing FR: Series 1 - S1E1" in report[1]

def test_trigger_wanted_search(mock_bazarr_api):
    import maintain_library
    _, mock_post = mock_bazarr_api
    
    maintain_library.trigger_wanted_search()
    assert mock_post.call_count == 2 # Movies and Series
    
def test_main_flow(mock_bazarr_api, mocker):
    import maintain_library
    mock_get, mock_post = mock_bazarr_api
    
    # Mock sequence of events
    # update_library post
    # wait_for_tasks get (empty)
    # trigger_wanted_search post (x2)
    # wait_for_tasks get (empty)
    # trigger_audio_sync post (run_task subsync)
    # wait_for_tasks get (empty)
    # get_integrity_report get (movies), get (series), get (episodes if series)
    
    mock_post.return_value = MagicMock(status_code=200)
    mock_get.return_value = MagicMock(json=lambda: []) # All empty/completed
    mocker.patch('time.sleep')
    
    # To avoid side_effect complexity for this flow, we just ensure it doesn't crash
    maintain_library.main()
    assert mock_post.called
    assert mock_get.called
