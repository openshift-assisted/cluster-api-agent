#!/usr/bin/env python3

import os
import sys
import subprocess
import logging
from datetime import datetime

script_dir = os.path.dirname(os.path.abspath(__file__))
project_root = os.path.dirname(script_dir)
if project_root not in sys.path:
    sys.path.insert(0, project_root)


from hack.release_candidates_repository import ReleaseCandidatesRepository
from hack.shared_types import SnapshotCollection, Snapshot

logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger("test-runner")

RELEASE_CANDIDATES_FILE = os.path.join(
    os.path.dirname(os.path.dirname(__file__)), "release-candidates.yaml"
)


def find_pending_snapshot(data: SnapshotCollection) -> tuple[str, Snapshot] | None:
    for snapshot in data["snapshots"]:
        if "metadata" in snapshot and snapshot["metadata"].get("status") == "pending":
            return snapshot["metadata"].get("id"), snapshot
    return None, None


def export_component_variables(snapshot: Snapshot) -> None:
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
        repo_name: str = repo.removeprefix("https://github.com/")
        image_url: str = component["image_url"]
        env_var = component_env_map.get(repo_name)
        if not env_var:
            logger.warning(
                f"No environment variable mapping for component: {component}"
            )
            continue

        if (
            repo == "https://github.com/kubernetes-sigs/cluster-api"
            or repo == "https://github.com/metal3-io/cluster-api-provider-metal3"
        ):
            value = component["ref"]
        else:
            value = component["image_url"]

        if repo_name == "openshift/assisted-installer":
            if "controller" in image_url.lower():
                env_var = "ASSISTED_INSTALLER_CONTROLLER_IMAGE"
            else:
                env_var = "ASSISTED_INSTALLER_IMAGE"

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


def update_snapshot_status(snapshot_id: str, success: bool) -> None:
    try:
        repo = ReleaseCandidatesRepository(RELEASE_CANDIDATES_FILE)
        updated = repo.update_status(
            snapshot_id,
            "successful" if success else "failed",
            datetime.now().isoformat(),
        )

        if updated:
            logger.info(
                f"Updated snapshot {snapshot_id} status to {'successful' if success else 'failed'}"
            )
        else:
            logger.error(f"Snapshot with ID {snapshot_id} not found")
    except Exception as e:
        logger.error(f"Error updating snapshot status: {e}")


def main() -> None:
    try:
        logger.info("Starting test runner")
        repo = ReleaseCandidatesRepository(RELEASE_CANDIDATES_FILE)
        pending_candidates = repo.find_by_status("pending")

        if not pending_candidates:
            logger.info("No pending snapshots to process")
            return

        candidate = pending_candidates[0]
        snapshot_id = candidate.metadata["id"]

        logger.info(f"Processing pending snapshot {snapshot_id}")
        export_component_variables(candidate.to_dict())

        success = run_ansible_tests()
        update_snapshot_status(snapshot_id, success)

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
