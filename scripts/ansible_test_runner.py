#!/usr/bin/env python3

import os
import sys
import subprocess
import logging
import yaml
from datetime import datetime
from shared_types import ReleaseCandidates, SnapshotInfo

logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger("test-runner")

RELEASE_CANDIDATES_FILE = "release-candidates.yaml"


def read_release_candidates() -> ReleaseCandidates:
    try:
        with open(RELEASE_CANDIDATES_FILE, "r") as f:
            data = yaml.safe_load(f)
        if not data or "snapshots" not in data:
            logger.error(f"Invalid format in {RELEASE_CANDIDATES_FILE}")
            sys.exit(1)
        return data
    except Exception as e:
        logger.error(f"Error reading {RELEASE_CANDIDATES_FILE}: {e}")
        sys.exit(1)


def find_pending_snapshot(data: ReleaseCandidates) -> tuple[str, SnapshotInfo] | None:
    for snapshot in data["snapshots"]:
        if "metadata" in snapshot and snapshot["metadata"].get("status") == "pending":
            return snapshot["metadata"].get("generated_at"), snapshot
    return None, None

def export_component_variables(snapshot: SnapshotInfo) -> None:
    env_vars = {}
    component_env_map = {
        "kubernetes-sigs/cluster-api": "CAPI_VERSION",
        "metal3-io/cluster-api-provider-metal3": "CAPM3_VERSION",
        "openshift/assisted-service": "ASSISTED_SERVICE_IMAGE",
        "openshift/assisted-image-service": "ASSISTED_IMAGE_SERVICE_IMAGE",
        "openshift/assisted-installer-agent": "ASSISTED_INSTALLER_AGENT_IMAGE",
        "openshift/assisted-installer-controller": "ASSISTED_INSTALLER_CONTROLLER_IMAGE",
        "openshift/assisted-installer": "ASSISTED_INSTALLER_IMAGE",
    }
    for component in snapshot["components"]:
        repo = component["repository"]
        env_var = component_env_map.get(repo)
        if not env_var:
            logger.warning(
                f"No environment variable mapping for component: {component}"
            )
            continue

        if (
            repo == "kubernetes-sigs/cluster-api"
            or repo == "metal3-io/cluster-api-provider-metal3"
        ):
            value = component["ref"]
        else:
            value = component["image_url"]

        if value:
            env_vars[env_var] = value
            os.environ[env_var] = value
            logger.info(f"Exported {env_var}={value}")


def run_ansible_tests() -> bool:
    try:
        logger.info("Running Ansible tests...")
        ansible_cmd = [
            "ansible-playbook",
            "test/ansible/run_test.yaml",
            "-i",
            "test/ansible/inventory.yaml",
        ]
        result = subprocess.run(
            ansible_cmd, check=False, capture_output=False, text=True
        )

        if result.returncode == 0:
            logger.info("Ansible tests passed successfully")
            return True
        else:
            logger.error(f"Ansible tests failed with return code {result.returncode}")
            return False

    except Exception as e:
        logger.error(f"Error running Ansible tests: {e}")
        return False

def update_snapshot_status(timestamp: str, success: bool) -> None:
    try:
        data: ReleaseCandidates = read_release_candidates()
        
        for snapshot in data["snapshots"]:
            if snapshot.get("metadata", {}).get("generated_at") == timestamp:
                if "metadata" not in snapshot:
                    snapshot["metadata"] = {}
                    
                snapshot["metadata"]["status"] = "successful" if success else "failed"
                snapshot["metadata"]["tested_at"] = datetime.now().isoformat()
                
                with open(RELEASE_CANDIDATES_FILE, "w") as f:
                    yaml.dump(data, f, default_flow_style=False)
                
                logger.info(f"Updated snapshot from {timestamp} status to {'successful' if success else 'failed'}")
                return
                
        logger.error(f"Snapshot with timestamp {timestamp} not found")
    except Exception as e:
        logger.error(f"Error updating snapshot status: {e}")


def main() -> None:
    try:
        logger.info("Starting test runner")
        data: ReleaseCandidates = read_release_candidates()

        timestamp, snapshot = find_pending_snapshot(data)
        if snapshot is None:
            logger.info("No pending snapshots to process")
            return

        logger.info(f"Processing pending snapshot from {timestamp}")
        export_component_variables(snapshot)

        success = run_ansible_tests()
        update_snapshot_status(timestamp, success)

        if success:
            logger.info("Snapshot processed successfully")
        else:
            logger.error("Snapshot testing failed")
            sys.exit(1)

    except Exception as e:
        logger.error(f"Test runner failed: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
