import pytest
from unittest.mock import MagicMock, patch

import sys
import os

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from scripts.tag_reconciler import TagReconciler


@pytest.fixture
def tag_reconciler():
    with patch("scripts.tag_reconciler.get_github_client") as mock_get_client:
        mock_get_client.return_value = MagicMock()
        reconciler = TagReconciler()
        return reconciler


def test_extract_base_repo(tag_reconciler):
    assert tag_reconciler.extract_base_repo("openshift/repo") == "openshift/repo"
    assert (
        tag_reconciler.extract_base_repo("openshift/repo/component") == "openshift/repo"
    )


def test_tag_exists(tag_reconciler):
    tag_reconciler.github_client.get_repo.return_value = repo_mock = MagicMock()

    tag_reconciler.tag_exists("test/repo", "v1.0.0")
    repo_mock.get_git_ref.assert_called_with("tags/v1.0.0")

    # test tag doesn't exist case
    repo_mock.get_git_ref.side_effect = Exception("Not found")
    assert tag_reconciler.tag_exists("test/repo", "v1.0.0") is False


def test_create_tag(tag_reconciler):
    tag_reconciler.github_client.get_repo.return_value = repo_mock = MagicMock()
    tag_mock = MagicMock()
    tag_mock.sha = "tag_sha"
    repo_mock.create_git_tag.return_value = tag_mock

    assert tag_reconciler.create_tag("test/repo", "commit_sha", "v1.0.0") is True
    repo_mock.create_git_tag.assert_called_with(
        tag="v1.0.0",
        message="Version v1.0.0 - Tagged by CAPBCOA CI",
        object="commit_sha",
        type="commit",
    )
    repo_mock.create_git_ref.assert_called_with("refs/tags/v1.0.0", "tag_sha")


def test_reconcile_tag(tag_reconciler):
    tag_reconciler.tag_exists = MagicMock(return_value=False)
    tag_reconciler.create_tag = MagicMock(return_value=True)

    version = {
        "name": "v1.0.0",
        "repositories": {"openshift/repo": {"commit_sha": "abc123"}},
    }

    assert tag_reconciler.reconcile_tag(version) is True
    tag_reconciler.create_tag.assert_called_with("openshift/repo", "abc123", "v1.0.0")
