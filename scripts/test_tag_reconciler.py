import pytest
from unittest.mock import MagicMock, patch, mock_open


import sys
import os

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from scripts.shared_types import VersionsFile
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
        "components": [
            {
                "repository": "openshift/repo",
                "ref": "abc123"
            }
        ]
    }

    assert tag_reconciler.reconcile_tag(version) is True
    tag_reconciler.create_tag.assert_called_with("openshift/repo", "abc123", "v1.0.0")

def test_read_versions_file_success(tag_reconciler):
    yaml_content = """
    versions:
      - name: v0.0.1
        components:
          - repository: kubernetes-sigs/cluster-api
            ref: v1.9.5
            image_url:
          - repository: metal3-io/cluster-api-provider-metal3
            ref: v1.9.3
            image_url:
          - repository: openshift/assisted-service/assisted-service
            ref: 76d29d2a7f0899dcede9700fc88fcbad37b6ccca
            image_url: quay.io/edge-infrastructure/assisted-service:latest-76d29d2a7f0899dcede9700fc88fcbad37b6ccca
          - repository: openshift/assisted-image-service/assisted-image-service
            ref: 2249c85d05600191b24e93dd92e733d49a1180ec
            image_url: quay.io/edge-infrastructure/assisted-image-service:latest-2249c85d05600191b24e93dd92e733d49a1180ec
          - repository: openshift/assisted-installer-agent/assisted-installer-agent
            ref: cfe93a9779dea6ad2a628280b40071d23f3cb429
            image_url: quay.io/edge-infrastructure/assisted-installer-agent:latest-cfe93a9779dea6ad2a628280b40071d23f3cb429
          - repository: openshift/assisted-installer/assisted-installer-controller
            ref: c389a38405383961d26191799161c86127451635
            image_url: quay.io/edge-infrastructure/assisted-installer-controller:latest-c389a38405383961d26191799161c86127451635
          - repository: openshift/assisted-installer/assisted-installer
            ref: c389a38405383961d26191799161c86127451635
            image_url: quay.io/edge-infrastructure/assisted-installer:latest-c389a38405383961d26191799161c86127451635
    """
    
    with patch("builtins.open", mock_open(read_data=yaml_content)):
        with patch("os.path.exists", return_value=True):
            result: VersionsFile = tag_reconciler.read_versions_file()
            
    assert result is not None
    assert "versions" in result
    assert len(result["versions"]) == 1
    assert result["versions"][0]["name"] == "v0.0.1"
    assert len(result["versions"][0]["components"]) == 7
