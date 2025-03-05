import os
import shutil
from unittest.mock import MagicMock, patch, call

import pytest
from core.models import Commit, Version
from core.services.tag_reconciliation_service import TagReconciliationService

ASSETS_DIR = os.path.join(os.path.dirname(os.path.dirname(__file__)), "assets")


@pytest.fixture
def versions_file(tmp_path):
    source_file = os.path.join(ASSETS_DIR, "versions.yaml")
    dest_file = tmp_path / "versions.yaml"
    shutil.copy(source_file, dest_file)
    return dest_file

@pytest.fixture
def mock_github():
    with patch("core.services.tag_reconciliation_service.GitHubClient") as mock:
        github_client = MagicMock()
        mock.return_value = github_client
        yield github_client

@pytest.fixture
def mock_version_repo():
    with patch("core.services.tag_reconciliation_service.VersionRepository") as mock:
        version_repo = MagicMock()
        mock.return_value = version_repo
        yield version_repo

@pytest.fixture
def service(versions_file, mock_version_repo, mock_github):
    with patch("core.services.tag_reconciliation_service.GitHubClient"), \
         patch("core.services.tag_reconciliation_service.VersionRepository"):
        svc = TagReconciliationService(versions_file)
        svc.logger = MagicMock()
        svc.github = mock_github
        svc.versions_repo = mock_version_repo
        svc.github.get_repo.return_value = MagicMock()
        svc.versions_repo.find_all.return_value = [
            Version(name="v1.0.0", commits=[
                Commit(repository="https://github.com/openshift/repo", ref="abc123")
            ])
        ]
        return svc


@pytest.fixture
def versions():
    return [
        Version(
            name="v1.0.0",
            commits=[
                Commit(repository="https://github.com/openshift/repo1", ref="abc123"),
                Commit(repository="https://github.com/openshift/repo2", ref="def456"),
                Commit(repository="https://github.com/kubernetes-sigs/repo3", ref="ghi789"),
            ]
        ),
        Version(
            name="v1.1.0",
            commits=[
                Commit(repository="https://github.com/openshift/repo1", ref="jkl012"),
                Commit(repository="https://github.com/openshift/repo2", ref="mno345"),
            ]
        ),
        Version(
            name="",  # Empty version name to test skipping
            commits=[
                Commit(repository="https://github.com/openshift/repo4", ref="pqr678"),
            ]
        )
    ]


def test_reconciler_creates_tag(service):
    service.tag_exists = MagicMock(return_value=False)
    service.create_tag = MagicMock()
    service.run()
    service.create_tag.assert_called_once()

def test_reconciler_skips_existing_tag(service):
    service.tag_exists = MagicMock(return_value=True)
    service.create_tag = MagicMock()
    service.run()
    service.create_tag.assert_not_called()


def test_run_happy_path_complete(service, mock_github, mock_version_repo, versions):
    # Setup
    mock_version_repo.find_all.return_value = versions
    
    # Configure tag_exists to return False (tags don't exist) for all repos
    service.tag_exists = MagicMock(return_value=False)
    service.create_tag = MagicMock()
    
    # Create mock GitHub repos
    mock_repo1 = MagicMock()
    mock_repo2 = MagicMock()
    
    # Setup get_repo to return the appropriate repo
    def get_repo_side_effect(name):
        if name == "openshift/repo1":
            return mock_repo1
        elif name == "openshift/repo2":
            return mock_repo2
        else:
            return MagicMock()
            
    mock_github.get_repo.side_effect = get_repo_side_effect
    
    # Run the service
    service.run()
    
    # Verify the right repos were checked for existing tags
    expected_tag_exists_calls = [
        call("openshift/repo1", "v1.0.0"),
        call("openshift/repo2", "v1.0.0"),
        call("openshift/repo1", "v1.1.0"),
        call("openshift/repo2", "v1.1.0"),
    ]
    service.tag_exists.assert_has_calls(expected_tag_exists_calls, any_order=True)
    
    # Verify that tags were created
    expected_create_tag_calls = [
        call("openshift/repo1", "abc123", "v1.0.0"),
        call("openshift/repo2", "def456", "v1.0.0"),
        call("openshift/repo1", "jkl012", "v1.1.0"),
        call("openshift/repo2", "mno345", "v1.1.0"),
    ]
    service.create_tag.assert_has_calls(expected_create_tag_calls, any_order=True)
    
    # verify that non-openshift repos were skipped
    for call_args in service.tag_exists.call_args_list:
        repo = call_args[0][0]
        assert repo.startswith("openshift/"), f"Non-openshift repo {repo} was checked"
    
    # verify that empty version names were skipped
    assert service.logger.warning.called
    empty_version_warning_found = any(
        "Skipping version without name" in str(args) 
        for args, _ in service.logger.warning.call_args_list
    )
    assert empty_version_warning_found, "Warning about empty version name not logged"



def test_run_with_existing_tags(service, mock_version_repo, versions):
    mock_version_repo.find_all.return_value = versions[:1]  # just use first version
    
    # tag exists for repo1, not for repo2
    def tag_exists_side_effect(repo, tag):
        return repo == "openshift/repo1"  
        
    service.tag_exists = MagicMock(side_effect=tag_exists_side_effect)
    service.create_tag = MagicMock()
    
    service.run()
    
    # assert tag creation was only called for repo2 (not repo1)
    service.create_tag.assert_called_once_with("openshift/repo2", "def456", "v1.0.0")

def test_run_with_tag_creation_failure(service, mock_version_repo, versions):
    mock_version_repo.find_all.return_value = versions[:1]
    
    service.tag_exists = MagicMock(return_value=False)
    
    def create_tag_side_effect(repo, ref, tag):
        if repo == "openshift/repo2":
            raise Exception(f"Failed to create tag {tag} on {repo}")
            
    service.create_tag = MagicMock(side_effect=create_tag_side_effect)
    
    with pytest.raises(Exception, match="Failed to create tag v1.0.0 on openshift/repo2"):
        service.run()
    
    service.create_tag.assert_any_call("openshift/repo1", "abc123", "v1.0.0")

def test_create_tag_implementation(service, mock_github):
    # Bypass our mocking and test actual implementation
    service.create_tag = TagReconciliationService.create_tag.__get__(service, TagReconciliationService)
    
    # Setup mock repo
    mock_repo = MagicMock()
    mock_tag_obj = MagicMock()
    mock_tag_obj.sha = "tag_sha_123"
    mock_repo.create_git_tag.return_value = mock_tag_obj
    mock_github.get_repo.return_value = mock_repo
    
    service.create_tag("openshift/repo1", "commit_sha_123", "v1.0.0")
    
    mock_repo.create_git_tag.assert_called_once_with(
        tag="v1.0.0", 
        message="Tagged by CI", 
        object="commit_sha_123", 
        type="commit"
    )
    mock_repo.create_git_ref.assert_called_once_with(
        "refs/tags/v1.0.0", 
        "tag_sha_123"
    )

def test_tag_exists_implementation(service, mock_github):
    service.tag_exists = TagReconciliationService.tag_exists.__get__(service, TagReconciliationService)
    
    mock_repo = MagicMock()
    mock_github.get_repo.return_value = mock_repo
    
    mock_repo.get_git_ref.return_value = MagicMock()
    assert service.tag_exists("openshift/repo1", "v1.0.0") is True
    mock_repo.get_git_ref.assert_called_with("tags/v1.0.0")
    
    mock_repo.get_git_ref.side_effect = Exception("Not found")
    assert service.tag_exists("openshift/repo1", "v1.0.0") is False

def test_run_no_versions(service, mock_version_repo):
    mock_version_repo.find_all.return_value = []
    
    service.tag_exists = MagicMock()
    service.create_tag = MagicMock()
    
    service.run()
    
    service.tag_exists.assert_not_called()
    service.create_tag.assert_not_called()


def test_tag_already_exists_detailed(service, mock_github, mock_version_repo, versions):
    mock_version_repo.find_all.return_value = versions[:1]  # Just use the first version (v1.0.0)
    
    # Setup GitHub repo responses
    repo1 = MagicMock()
    repo2 = MagicMock()
    
    def get_repo_side_effect(name):
        if name == "openshift/repo1":
            return repo1
        elif name == "openshift/repo2":
            return repo2
        else:
            return MagicMock()
    
    mock_github.get_repo.side_effect = get_repo_side_effect
    
    def get_git_ref_side_effect(ref_path):
        if ref_path == "tags/v1.0.0" and repo1 == mock_github.get_repo("openshift/repo1"):
            return MagicMock()
        raise Exception("Tag not found")
    
    repo1.get_git_ref.side_effect = get_git_ref_side_effect
    repo2.get_git_ref.side_effect = Exception("Tag not found")
    
    original_create_tag = service.create_tag
    create_tag_calls = []
    
    def spy_create_tag(repo, ref, tag):
        create_tag_calls.append((repo, ref, tag))
        return original_create_tag(repo, ref, tag)
    
    service.create_tag = MagicMock(side_effect=spy_create_tag)
    
    service.run()
    
    # assert that repo1 tag was NOT created (already exists), but repo2 tag was created
    assert len(create_tag_calls) == 1
    assert create_tag_calls[0][0] == "openshift/repo2"  # Only repo2 should get a tag
    assert create_tag_calls[0][2] == "v1.0.0"  # With version v1.0.0
    
    repo1.get_git_ref.assert_called_with("tags/v1.0.0")
    repo2.get_git_ref.assert_called_with("tags/v1.0.0")
    
    log_messages = [args[0] for args, _ in service.logger.info.call_args_list]
    repo2_message_found = any("Created tag v1.0.0 on openshift/repo2" in msg for msg in log_messages)
    assert repo2_message_found

def test_tag_creation_failure_detailed(service, mock_github, mock_version_repo, versions):
    mock_version_repo.find_all.return_value = versions[:1]  # Just use the first version (v1.0.0)
    
    # no tags exist
    mock_repo = MagicMock()
    mock_repo.get_git_ref.side_effect = Exception("Tag not found")
    mock_github.get_repo.return_value = mock_repo
    
    # setup create_git_tag to fail with a specific error for repo2
    def create_git_tag_side_effect(tag, message, object, type):
        if tag == "v1.0.0" and "repo2" in str(mock_github.get_repo.call_args[0][0]):
            raise Exception("GitHub API error: Reference already exists")
        return MagicMock(sha="fake_sha")
    
    mock_repo.create_git_tag.side_effect = create_git_tag_side_effect
    
    with pytest.raises(Exception) as excinfo:
        service.run()
    
    # verify the error contains both the repo name and the original error
    error_message = str(excinfo.value)
    assert "Failed to create tag v1.0.0 on openshift/repo2" in error_message
    assert "GitHub API error: Reference already exists" in error_message
    
    # verify that tag creation was attempted for repo1 before failing on repo2
    mock_repo.create_git_tag.assert_called_with(
        tag="v1.0.0", 
        message="Tagged by CI", 
        object="def456",  # This is the ref for repo2 in our test data
        type="commit"
    )
