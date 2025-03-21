#!/usr/bin/env python3

import re
import os
import sys
import logging
from ruamel.yaml import YAML

script_dir = os.path.dirname(os.path.abspath(__file__))
project_root = os.path.dirname(script_dir)
if project_root not in sys.path:
    sys.path.insert(0, project_root)


from hack.github_auth import get_github_client
from hack.shared_types import Version, VersionsCollection

logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger("tag-reconciler")

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PROJECT_ROOT = os.path.dirname(SCRIPT_DIR)
VERSIONS_FILE = os.path.join(PROJECT_ROOT, "versions.yaml")

yaml = YAML(typ="rt")
yaml.default_flow_style = False
yaml.explicit_start = False
yaml.preserve_quotes = True
yaml.width = 4096
yaml.indent(mapping=2, sequence=2, offset=0)


class TagReconciler:
    def __init__(self):
        self.github_client = get_github_client()

    def read_versions_file(self) -> VersionsCollection:
        if not os.path.exists(VERSIONS_FILE):
            logger.error(f"{VERSIONS_FILE} does not exist")
            sys.exit(1)

        try:
            with open(VERSIONS_FILE, "r") as f:
                data = yaml.load(f)
            return data or {}
        except Exception as e:
            raise Exception(f"Error reading {VERSIONS_FILE}: {e}")

    def tag_exists(self, repo: str, tag: str) -> bool:
        try:
            github_repo = self.github_client.get_repo(repo)
            github_repo.get_git_ref(f"tags/{tag}")
            return True
        except Exception as e:
            logger.error(f"Error checking tag {tag} for {repo}: {e}")
            return False

    def create_tag(self, repo: str, commit_sha: str, tag: str) -> bool:
        message = f"Version {tag} - Tagged by CAPBCOA CI"

        try:
            github_repo = self.github_client.get_repo(repo)
            tag_obj = github_repo.create_git_tag(
                tag=tag,
                message=message,
                object=commit_sha,
                type="commit"
            )

            github_repo.create_git_ref(f"refs/tags/{tag}", tag_obj.sha)
            logger.info(f"Created tag {tag} for {repo}")
            return True
        except Exception as e:
            logger.error(f"Error creating tag {tag} for {repo}: {e}")
            return False

    def extract_base_repo(self, full_repo_name: str) -> str:
        parts = full_repo_name.split("/")
        if len(parts) > 2:
            return f"{parts[0]}/{parts[1]}"
        return full_repo_name

    def ensure_tag_exists(
        self,
        ref: str,
        repo_name: str,
        version_name: str,
    ) -> bool:
        base_repo = self.extract_base_repo(repo_name)
        if self.tag_exists(base_repo, version_name):
            logger.info(f"Tag {version_name} already exists for {base_repo}")
            return True
        else:
            success = self.create_tag(base_repo, ref, version_name)
            if not success:
                logger.error(f"Failed to create tag {version_name} for {base_repo}")
                return False
            return True

    def reconcile_tag(self, version: Version) -> bool:
        version_name: str = version["name"]
        if not version_name:
            logger.error("Version entry missing 'name' field")
            return False

        logger.info(f"Processing version {version_name}")
        success = True

        components = version["components"]
        if not components:
            logger.warning(f"No repositories found for version {version_name}")
            return True

        for component in components:
            repo_name: str = component["repository"].removeprefix("https://github.com/")
            ref: str = component["ref"]

            if not re.match(r"^openshift/", repo_name) or not ref:
                logger.warning(f"Skipping repository {repo_name}")
                continue

            success &= self.ensure_tag_exists(ref, repo_name, version_name)

        return success

    def reconcile_tags(self) -> bool:
        versions_data: VersionsCollection = self.read_versions_file()
        versions: list[Version] = versions_data["versions"]

        logger.info(f"Processing {len(versions)} versions")
        success = True

        for version_entry in versions:
            if not self.reconcile_tag(version_entry):
                success = False

        return success


def main() -> None:
    try:
        logger.info("Starting tag reconciliation")
        reconciler = TagReconciler()
        success = reconciler.reconcile_tags()

        if not success:
            logger.error("Tag reconciliation completed with errors")
            sys.exit(1)

    except Exception as e:
        logger.error(f"Error in tag reconciliation: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
