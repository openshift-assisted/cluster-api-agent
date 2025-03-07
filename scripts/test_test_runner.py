from unittest.mock import patch
import os

import sys
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from scripts.test_runner import find_pending_snapshot, export_component_variables, run_ansible_tests


def test_find_pending_snapshot():
    data = {
        "snapshots": [
            {"metadata": {"status": "successful"}},
            {"metadata": {"status": "pending"}},
            {"metadata": {"status": "failed"}}
        ]
    }
    
    # Should find the second snapshot
    index, snapshot = find_pending_snapshot(data)
    assert index == 1
    assert snapshot["metadata"]["status"] == "pending"
    
    # No pending snapshots
    data = {"snapshots": [{"metadata": {"status": "successful"}}]}
    index, snapshot = find_pending_snapshot(data)
    assert index is None
    assert snapshot is None


def test_export_component_variables():
    snapshot = {
        "versions": {
            "kubernetes-sigs/cluster-api": {"version": "v1.0.0"},
            "openshift/assisted-service/assisted-service": {
                "image_url": "quay.io/edge-infrastructure/assisted-service:latest-abc123"
            }
        }
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
