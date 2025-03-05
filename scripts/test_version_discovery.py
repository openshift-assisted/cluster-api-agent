import pytest
from unittest.mock import MagicMock, patch

import sys
import os
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from scripts.version_discovery import ComponentScanner


@pytest.fixture
def component_scanner():
    with patch('scripts.version_discovery.get_github_client') as mock_get_client:
        mock_get_client.return_value = MagicMock()
        scanner = ComponentScanner()
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


def test_process_repository_clusterctl(component_scanner):
    component_scanner.find_latest_release_version = MagicMock(return_value=("v1.0.0", "main"))
    
    config = {
        "tag_format": "clusterctl",
        "version_prefix": "v"
    }
    
    result = component_scanner.process_repository("org/repo", config)
    assert "org/repo" in result
    assert result["org/repo"]["version"] == "v1.0.0"
    assert result["org/repo"]["commit_sha"] == "main"


def test_process_repository_commit(component_scanner):
    component_scanner.find_latest_commit_with_image = MagicMock(return_value=("abc123", "latest-abc123"))
    
    config = {
        "tag_format": "commit",
        "images": ["quay.io/namespace/image"]
    }
    
    result = component_scanner.process_repository("org/repo", config)
    assert "org/repo/image" in result
    assert result["org/repo/image"]["version"] == "abc123"
    assert result["org/repo/image"]["image_url"] == "quay.io/namespace/image:latest-abc123"
