#!/usr/bin/env python3

import sys
import requests
import os
from ruamel.yaml import YAML
from datetime import datetime
from typing import Any
import logging
from concurrent.futures import ThreadPoolExecutor
from github_auth import get_github_client
from shared_types import ReleaseCandidates, RepositoryConfig, RepositoryInfo, SnapshotInfo

logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger("component-scanner")


yaml = YAML(typ="rt")
yaml.default_flow_style = False
yaml.explicit_start = False
yaml.preserve_quotes = True
yaml.width = 4096
yaml.indent(mapping=2, sequence=2, offset=0)


REPOSITORIES: dict[str, RepositoryConfig] = {
    "kubernetes-sigs/cluster-api": {
        "images": None,
        "version_prefix": "v",
    },
    "metal3-io/cluster-api-provider-metal3": {
        "images": None,
        "version_prefix": "v",
    },
    "openshift/assisted-service": {
        "images": ["quay.io/edge-infrastructure/assisted-service"],
    },
    "openshift/assisted-image-service": {
        "images": ["quay.io/edge-infrastructure/assisted-image-service"],
    },
    "openshift/assisted-installer-agent": {
        "images": ["quay.io/edge-infrastructure/assisted-installer-agent"],
    },
    "openshift/assisted-installer": {
        "images": [
            "quay.io/edge-infrastructure/assisted-installer-controller",
            "quay.io/edge-infrastructure/assisted-installer",
        ],
    },
}


class ComponentScanner:
    def __init__(self):
        self.github_client = get_github_client()

    def get_repository_commits(
        self, repo: str, max_commits: int = 20
    ) -> list[dict[str, str]]:
        try:
            github_repo = self.github_client.get_repo(repo)
            commits: list[dict[str, str]] = []
            for commit in github_repo.get_commits()[:max_commits]:
                commits.append({"sha": commit.sha})
            return commits
        except Exception as e:
            logger.error(f"Error fetching commits for {repo}: {e}")
            raise

    def get_repository_releases(
        self, repo: str, max_releases: int = 10
    ) -> list[dict[str, Any]]:
        try:
            github_repo = self.github_client.get_repo(repo)
            releases = []
            for release in github_repo.get_releases()[:max_releases]:
                releases.append(
                    {"tag_name": release.tag_name, "prerelease": release.prerelease}
                )
            return releases
        except Exception as e:
            logger.error(f"Error fetching releases for {repo}: {e}")
            raise

    def check_image_exists(self, image: str, tag: str) -> bool:
        if image.startswith("quay.io/"):
            registry_url = "https://quay.io"
            parts = image.replace("quay.io/", "").split("/")
        else:
            logger.warning(f"Unsupported registry for image: {image}")
            return False

        if len(parts) > 2:
            namespace = parts[0]
            repository = "/".join(parts[1:])
        else:
            namespace, repository = parts

        url = f"{registry_url}/v2/{namespace}/{repository}/manifests/{tag}"
        headers = {"Accept": "application/vnd.docker.distribution.manifest.v2+json"}

        try:
            response = requests.head(url, headers=headers)
            exists = response.status_code == 200
            logger.debug(
                f"Image {image}:{tag} exists: {exists} (status: {response.status_code})"
            )
            return exists
        except requests.RequestException as e:
            logger.error(f"Error checking image {image}:{tag}: {e}")
            return False

    def find_latest_release_version(
        self, repo: str, version_prefix: str
    ) -> tuple[str, str] | None:
        try:
            github_repo = self.github_client.get_repo(repo)
            for release in github_repo.get_releases():
                if (
                    release.tag_name.startswith(version_prefix)
                    and not release.prerelease
                ):
                    logger.info(f"Found release {release.tag_name} for {repo}")
                    return release.tag_name, release.target_commitish
            return None
        except Exception as e:
            logger.error(f"Error finding latest release for {repo}: {e}")
            raise

    def find_latest_commit_with_image(
        self, repo: str, image: str
    ) -> tuple[str, str] | None:
        try:
            github_repo = self.github_client.get_repo(repo)
            commits = github_repo.get_commits()[:20]

            for commit in commits:
                sha = commit.sha
                tag = f"latest-{sha}"

                if self.check_image_exists(image, tag):
                    logger.info(f"Found matching image for {repo} at commit {sha[:8]}")
                    return sha, tag

            logger.warning(f"No commits found with matching images for {repo}/{image}")
            return None
        except Exception as e:
            logger.error(f"Error finding commit with image for {repo}: {e}")
            raise

    def scan_components(self) -> list[RepositoryInfo]:
        results: list[RepositoryInfo] = []
        with ThreadPoolExecutor(max_workers=10) as executor:
            future_to_repo = {
                executor.submit(self.process_repository, repo, config): repo
                for repo, config in REPOSITORIES.items()
            }
            for future in future_to_repo:
                repo = future_to_repo[future]
                try:
                    repo_results = future.result()
                    if repo_results:
                        results.extend(repo_results)
                except Exception as e:
                    logger.error(f"Error processing {repo}: {e}")

        return results

    def process_repository(
        self, repo: str, config: RepositoryConfig
    ) -> list[RepositoryInfo]:
        repo_results: list[RepositoryInfo] = []
        version_prefix = config.get("version_prefix", "v")
        images = config.get("images", [])

        version_info = None
        try:
            version_info = self.find_latest_release_version(repo, version_prefix)
        except Exception as e:
            logger.error(f"Error finding latest release for {repo}: {e}")
            
        if version_info:
            tag_name, _ = version_info
            repo_results.append({
                "repository": repo,
                "ref": tag_name,
                "image_url": None,
            })
        elif images:
            for image in images:
                component_name = f"{repo}/{image.split('/')[-1]}"
                logger.info(f"Checking {component_name} latest commit")

                commit_info = None
                try:
                    commit_info = self.find_latest_commit_with_image(repo, image)
                except Exception as e:
                    logger.error(f"Error finding commit with image for {repo}/{image}: {e}")
                    
                if commit_info:
                    commit_sha, tag = commit_info
                    repo_results.append({
                        "repository": repo,
                        "ref": commit_sha,
                        "image_url": f"{image}:{tag}",
                    })
                else:
                    logger.warning(f"No images found for {component_name}")
        else:
            logger.warning(f"No releases or tags found for {repo} and no images configured")

        return repo_results

    @staticmethod
    def save_to_release_candidates(
        results: list[RepositoryInfo], filename: str = "release-candidates.yaml"
    ) -> None:
        timestamp = datetime.now().isoformat()
        new_scan: SnapshotInfo = {
            "metadata": {"generated_at": timestamp, "status": "pending"},
            "components": results,
        }

        root_data: ReleaseCandidates = {"snapshots": []}

        if os.path.exists(filename):
            try:
                with open(filename, "r") as f:
                    root_data = yaml.load(f) or {"snapshots": []}

                if "snapshots" not in root_data:
                    root_data["snapshots"] = []

                # compare with first element if list is not empty
                if (
                    root_data["snapshots"]
                    and ComponentScanner._components_equal(root_data["snapshots"][0].get("components", []), results)
                ):
                    logger.info("No changes detected, skipping update")
                    return
                root_data["snapshots"].insert(0, new_scan)
            except Exception as e:
                logger.warning(f"Error reading existing file, will create new: {e}")
                root_data = {"snapshots": [new_scan]}
        else:
            root_data = {"snapshots": [new_scan]}

        with open(filename, "w") as f:
            yaml.dump(root_data, f)

        logger.info(f"Saved results to {filename}")

    @staticmethod
    def _components_equal(components1, components2):
        set1: set = {(d.get("repository", ""), d.get("ref", ""), d.get("image_url", "")) for d in components1}
        set2: set = {(d.get("repository", ""), d.get("ref", ""), d.get("image_url", "")) for d in components2}
        return set1 == set2

def main() -> None:
    try:
        logger.info("Starting component scan")
        scanner = ComponentScanner()
        results = scanner.scan_components()
        if not results:
            logger.error("No components found")
            sys.exit(1)

        scanner.save_to_release_candidates(results)
        logger.info(f"Found {len(results)} components")

    except Exception as e:
        logger.error(f"Error scanning components: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
