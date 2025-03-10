from unittest.mock import patch, mock_open
import os

import sys

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from scripts.test_runner import find_pending_snapshot, export_component_variables, read_release_candidates, run_ansible_tests


def test_find_pending_snapshot():
    data = {
        "snapshots": [
            {"metadata": {"status": "successful", "generated_at": "2025-03-10T10:00:00.000000"}},
            {"metadata": {"status": "pending", "generated_at": "2025-03-10T11:00:00.000000"}},
            {"metadata": {"status": "failed", "generated_at": "2025-03-10T12:00:00.000000"}}
        ]
    }
    
    # shoudl find the second snapshot
    timestamp, snapshot = find_pending_snapshot(data)
    assert timestamp == "2025-03-10T11:00:00.000000"
    assert snapshot["metadata"]["status"] == "pending"
    
    # no pending snapshots
    data = {"snapshots": [{"metadata": {"status": "successful", "generated_at": "2025-03-10T10:00:00.000000"}}]}
    timestamp, snapshot = find_pending_snapshot(data)
    assert timestamp is None
    assert snapshot is None

def test_export_component_variables():
    snapshot = {
        "components": [
            {
                "repository": "kubernetes-sigs/cluster-api",
                "ref": "v1.0.0",
                "image_url": None
            },
            {
                "repository": "openshift/assisted-service",
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

def test_read_release_candidates_success():
    yaml_content = """
    snapshots:
      - metadata:
          generated_at: '2025-03-10T10:32:04.642635'
          status: pending
        components:
          - repository: https://github.com/kubernetes-sigs/cluster-api
            ref: v1.9.5
            image_url:
          - repository: https://github.com/metal3-io/cluster-api-provider-metal3
            ref: v1.9.3
            image_url:
          - repository: assisted-service/assisted-service
            ref: 76d29d2a7f0899dcede9700fc88fcbad37b6ccca
            image_url: quay.io/edge-infrastructure/assisted-service:latest-76d29d2a7f0899dcede9700fc88fcbad37b6ccca
          - repository: assisted-image-service/assisted-image-service
            ref: 2249c85d05600191b24e93dd92e733d49a1180ec
            image_url: quay.io/edge-infrastructure/assisted-image-service:latest-2249c85d05600191b24e93dd92e733d49a1180ec
          - repository: assisted-installer-agent/assisted-installer-agent
            ref: cfe93a9779dea6ad2a628280b40071d23f3cb429
            image_url: quay.io/edge-infrastructure/assisted-installer-agent:latest-cfe93a9779dea6ad2a628280b40071d23f3cb429
          - repository: assisted-installer/assisted-installer-controller
            ref: c389a38405383961d26191799161c86127451635
            image_url: quay.io/edge-infrastructure/assisted-installer-controller:latest-c389a38405383961d26191799161c86127451635
          - repository: openshift/assisted-installer/assisted-installer
            ref: c389a38405383961d26191799161c86127451635
            image_url: quay.io/edge-infrastructure/assisted-installer:latest-c389a38405383961d26191799161c86127451635
    """
    
    with patch("builtins.open", mock_open(read_data=yaml_content)):
        result = read_release_candidates()
        
    assert result is not None
    assert "snapshots" in result
    assert len(result["snapshots"]) == 1
    assert "metadata" in result["snapshots"][0]
    assert result["snapshots"][0]["metadata"]["status"] == "pending"
    assert result["snapshots"][0]["metadata"]["generated_at"] == '2025-03-10T10:32:04.642635'
    assert "components" in result["snapshots"][0]
