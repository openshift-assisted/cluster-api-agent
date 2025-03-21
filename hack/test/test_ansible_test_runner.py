from unittest.mock import patch
import os

import sys

from hack.release_candidates_repository import ReleaseCandidatesRepository

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from hack.ansible_test_runner import find_pending_snapshot, export_component_variables, run_ansible_tests


def test_find_pending_snapshot():
    data = {
        "snapshots": [
            {"metadata": {"id": "123", "status": "successful", "generated_at": "2025-03-10T10:00:00.000000"}},
            {"metadata": {"id": "124", "status": "pending", "generated_at": "2025-03-10T11:00:00.000000"}},
            {"metadata": {"id": "125", "status": "failed", "generated_at": "2025-03-10T12:00:00.000000"}}
        ]
    }
    
    # shoudl find the second snapshot
    snapshot_id, snapshot = find_pending_snapshot(data)
    assert snapshot_id == "124"
    assert snapshot["metadata"]["status"] == "pending"
    
    # no pending snapshots
    data = {"snapshots": [{"metadata": {"id": "123", "status": "successful", "generated_at": "2025-03-10T10:00:00.000000"}}]}
    snapshot_id, snapshot = find_pending_snapshot(data)
    assert snapshot_id is None
    assert snapshot is None

def test_export_component_variables():
    snapshot = {
        "components": [
            {
                "repository": "https://github.com/kubernetes-sigs/cluster-api",
                "ref": "v1.0.0",
                "image_url": None
            },
            {
                "repository": "https://github.com/openshift/assisted-service",
                "ref": "abc123",
                "image_url": "quay.io/edge-infrastructure/assisted-service:latest-abc123"
            }
        ]
    }
    
    with patch.dict(os.environ, {}, clear=True):
        export_component_variables(snapshot)
        
        assert os.environ["CAPI_VERSION"] == "v1.0.0"
        assert os.environ["ASSISTED_SERVICE_IMAGE"] == "quay.io/edge-infrastructure/assisted-service:latest-abc123"

def test_run_ansible_tests():
    expected_cmd = [
        "ansible-playbook",
        "test/ansible/run_test.yaml",
        "-i",
        "test/ansible/inventory.yaml",
    ]
    
    with patch('subprocess.run') as mock_run:
        mock_run.return_value.returncode = 0
        assert run_ansible_tests() is True
        mock_run.assert_called_once_with(
            expected_cmd, 
            check=False, 
            capture_output=False, 
            text=True
        )
        
    # failure case
    with patch('subprocess.run') as mock_run:
        mock_run.return_value.returncode = 1
        assert run_ansible_tests() is False
        mock_run.assert_called_once_with(
            expected_cmd, 
            check=False, 
            capture_output=False, 
            text=True
        )

def get_test_release_candidates_yaml_file_path():
    return os.path.join(os.path.dirname(os.path.abspath(__file__)), "assets", "test_release_candidates.yaml")


def test_read_release_candidates_success():
    test_file_path = get_test_release_candidates_yaml_file_path()
    repo = ReleaseCandidatesRepository(test_file_path)
    result = repo.find_by_status("pending")

    assert result is not None
    assert len(result) == 1
    assert len(result[0].components) == 7

