#!/usr/bin/env python3

import re
import os
import sys
import logging
from typing import Any
from ruamel.yaml import YAML
from github_auth import get_github_client

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
    
    def read_versions_file(self) -> dict[str, Any]:
        if not os.path.exists(VERSIONS_FILE):
            logger.error(f"{VERSIONS_FILE} does not exist")
            sys.exit(1)

        try:
            with open(VERSIONS_FILE, "r") as f:
                data = yaml.load(f)
            return data or {}
        except Exception as e:
            logger.error(f"Error reading {VERSIONS_FILE}: {e}")
            return {}

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

    def reconcile_tag(self, version: dict[str, Any]) -> bool:
        version_name = version.get("name", "")
        if not version_name:
            logger.error("Version entry missing 'name' field")
            return False

        logger.info(f"Processing version {version_name}")
        success = True

        repositories = version.get("repositories", {})
        if not repositories:
            logger.warning(f"No repositories found for version {version_name}")
            return True

        for repo_name, repo_info in repositories.items():
            commit_sha = repo_info.get("commit_sha", "")

            if not re.match(r"^openshift/", repo_name) or not commit_sha:
                logger.warning(f"Skipping repository {repo_name}")
                continue

            base_repo = self.extract_base_repo(repo_name)
            if self.tag_exists(base_repo, version_name):
                logger.info(f"Tag {version_name} already exists for {base_repo}")
                continue
            if not self.create_tag(base_repo, commit_sha, version_name):
                logger.error(f"Failed to create tag {version_name} for {base_repo}")
                success = False

        return success

    def reconcile_tags(self) -> bool:
        versions_data = self.read_versions_file()
        versions = versions_data.get("versions", [])

        if not versions:
            logger.info("No versions to process")
            return True

        logger.info(f"Processing {len(versions)} versions")
        success = True

        for version in versions:
            if not self.reconcile_tag(version):
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
