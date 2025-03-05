import pytest
from unittest.mock import MagicMock, patch, mock_open

import sys
import os
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from version_discovery import ComponentScanner


@pytest.fixture
def component_scanner():
    with patch('version_discovery.get_github_client') as mock_get_client:
        mock_get_client.return_value = MagicMock()
        scanner = ComponentScanner()
        scanner.lock = MagicMock()
        return scanner

def test_check_image_exists(component_scanner):
    with patch('requests.head') as mock_head:
        mock_head.return_value.status_code = 200
        assert component_scanner.check_image_exists("quay.io/namespace/repo", "tag") is True
    
    with patch('requests.head') as mock_head:
        mock_head.return_value.status_code = 404
        assert component_scanner.check_image_exists("quay.io/namespace/repo", "tag") is False

def test_find_latest_release_version(component_scanner):
    mock_repo = MagicMock()
    component_scanner.github_client.get_repo.return_value = mock_repo
    
    release1 = MagicMock(tag_name="v1.0.0", target_commitish="main", prerelease=False)
    release2 = MagicMock(tag_name="v0.9.0", target_commitish="old", prerelease=False)

    # prerelease should be ignored!!
    release3 = MagicMock(tag_name="v1.1.0", target_commitish="dev", prerelease=True)  
    mock_repo.get_releases.return_value = [release1, release2, release3]
    
    result = component_scanner.find_latest_release_version("org/repo", "v")
    assert result == ("v1.0.0", "main")

def test_process_repository_tag(component_scanner):
    component_scanner.find_latest_release_version = MagicMock(return_value=("v1.0.0", "main"))
    
    config = {
        "version_prefix": "v"
    }
    
    result = component_scanner.process_repository("org/repo", config)
    assert result[0]["repository"] == "https://github.com/org/repo"
    assert result[0]["ref"] == "v1.0.0"

def test_process_repository_commit(component_scanner):
    component_scanner.find_latest_commit_with_image = MagicMock(return_value=("abc123", "latest-abc123"))
    
    config = {
        "images": ["quay.io/namespace/image"]
    }
    
    result = component_scanner.process_repository("org/repo", config)
    assert result[0]["repository"] == "https://github.com/org/repo"
    assert result[0]["ref"] == "abc123"
    assert result[0]["image_url"] == "quay.io/namespace/image:latest-abc123"

def test_save_to_release_candidates_success(component_scanner):
    sample_results = [
        {
            "repository": "https://github.com/org/repo",
            "ref": "v1.0.0",
            "image_url": None
        }
    ]
    
    mock_yaml_content = """
    snapshots:
      - metadata:
          generated_at: '2025-03-10T10:32:04.642635'
          status: successful
        components:
          - repository: org/repo
            ref: v0.9.0
            image_url: 
    """
    
    m = mock_open(read_data=mock_yaml_content)
    
    with patch("os.path.exists", return_value=True), patch("builtins.open", m):
        ComponentScanner.save_to_release_candidates(results=sample_results, filename="test.yaml")
    
    #verify that open was called for writing
    m.assert_called_with("test.yaml", "w")
    
    #verify that something was written
    handle = m()
    assert handle.write.called


def test_save_to_release_candidates_new_file(component_scanner):
    sample_results = [
        {
            "repository": "https://github.com/org/repo",
            "ref": "v1.0.0",
            "image_url": None
        }
    ]
    
    m = mock_open()
    
    with patch("os.path.exists", return_value=False), patch("builtins.open", m):
        ComponentScanner.save_to_release_candidates(sample_results, "test.yaml")
    
    m.assert_called_with("test.yaml", "w")
    
    handle = m()
    assert handle.write.called
