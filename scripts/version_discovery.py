#!/usr/bin/env python3

import sys
import requests
import os
from ruamel.yaml import YAML
from datetime import datetime
from typing import Any, TypedDict
import logging
from concurrent.futures import ThreadPoolExecutor
from github_auth import get_github_client

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


class ComponentInfo(TypedDict):
    commit_sha: str
    commit_url: str
    version: str
    image_url: str | None


class RepositoryConfig(TypedDict, total=False):
    images: list[str] | None
    tag_format: str
    version_prefix: str


REPOSITORIES: dict[str, RepositoryConfig] = {
    "kubernetes-sigs/cluster-api": {
        "images": None,
        "tag_format": "clusterctl",
        "version_prefix": "v",
    },
    "metal3-io/cluster-api-provider-metal3": {
        "images": None,
        "tag_format": "clusterctl",
        "version_prefix": "v",
    },
    "openshift/assisted-service": {
        "images": ["quay.io/edge-infrastructure/assisted-service"],
        "tag_format": "commit",
    },
    "openshift/assisted-image-service": {
        "images": ["quay.io/edge-infrastructure/assisted-image-service"],
        "tag_format": "commit",
    },
    "openshift/assisted-installer-agent": {
        "images": ["quay.io/edge-infrastructure/assisted-installer-agent"],
        "tag_format": "commit",
    },
    "openshift/assisted-installer": {
        "images": [
            "quay.io/edge-infrastructure/assisted-installer-controller",
            "quay.io/edge-infrastructure/assisted-installer",
        ],
        "tag_format": "commit",
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

    def scan_components(self) -> dict[str, ComponentInfo]:
        results: dict[str, ComponentInfo] = {}
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
                        results.update(repo_results)
                except Exception as e:
                    logger.error(f"Error processing {repo}: {e}")

        return results

    def process_repository(
        self, repo: str, config: RepositoryConfig
    ) -> dict[str, ComponentInfo]:
        repo_results: dict[str, ComponentInfo] = {}

        tag_format = config.get("tag_format", "commit")
        version_prefix = config.get("version_prefix", "v")
        images = config.get("images", [])

        # handle capi and capm3 tag format
        if tag_format == "clusterctl":
            logger.info(f"Checking {repo} using clusterctl-compatible versioning")

            version_info = self.find_latest_release_version(repo, version_prefix)

            if version_info:
                tag_name, commit_sha = version_info
                repo_results[repo] = {
                    "commit_sha": commit_sha,
                    "commit_url": f"https://github.com/{repo}/commit/{commit_sha}"
                    if commit_sha
                    else f"https://github.com/{repo}/releases/tag/{tag_name}",
                    "version": tag_name,
                    "image_url": None,
                }
            else:
                logger.warning(f"No releases or tags found for {repo}")

        # assisted tag format
        elif tag_format == "commit":
            if not images:
                logger.warning(f"No images configured for {repo}")
                return repo_results

            for image in images:
                component_name = f"{repo}/{image.split('/')[-1]}"
                logger.info(f"Checking {component_name} using commit-based tags")

                commit_info = self.find_latest_commit_with_image(repo, image)
                if commit_info:
                    commit_sha, tag = commit_info
                    repo_results[component_name] = {
                        "commit_sha": commit_sha,
                        "commit_url": f"https://github.com/{repo}/commit/{commit_sha}",
                        "version": tag.replace("latest-", ""),
                        "image_url": f"{image}:{tag}",
                    }
                else:
                    logger.warning(f"No commit-based images found for {component_name}")

        return repo_results

    @staticmethod
    def save_to_release_candidates(
        results: dict[str, ComponentInfo], filename: str = "release-candidates.yaml"
    ) -> None:
        timestamp = datetime.now().isoformat()
        new_scan = {
            "metadata": {"generated_at": timestamp, "status": "pending"},
            "versions": results,
        }

        root_data = {"snapshots": []}

        if os.path.exists(filename):
            try:
                with open(filename, "r") as f:
                    root_data = yaml.load(f) or {"snapshots": []}

                if "snapshots" not in root_data:
                    root_data["snapshots"] = []

                # compare with first element if list is not empty
                if (
                    root_data["snapshots"]
                    and root_data["snapshots"][0].get("versions") == results
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
