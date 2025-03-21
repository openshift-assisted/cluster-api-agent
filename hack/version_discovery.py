#!/usr/bin/env python3

import sys
import os
import requests
from typing import Any
import logging
from concurrent.futures import ThreadPoolExecutor

script_dir = os.path.dirname(os.path.abspath(__file__))
project_root = os.path.dirname(script_dir)
if project_root not in sys.path:
    sys.path.insert(0, project_root)


from hack.github_auth import get_github_client
from hack.shared_types import RepositoryConfig, RepositoryInfo
from hack.release_candidates_repository import ReleaseCandidate, ReleaseCandidatesRepository

logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger("component-scanner")


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
        version_prefix = config.get("version_prefix", "v")
        images = config.get("images", [])

        try:
            version_info = self.find_latest_release_version(repo, version_prefix)
            if version_info:
                tag_name, _ = version_info
                return [
                    {
                        "repository": f"https://github.com/{repo}",
                        "ref": tag_name,
                        "image_url": None,
                    }
                ]
        except Exception as e:
            logger.error(f"Error finding latest release for {repo}: {e}")

        if not images:
            logger.warning(
                f"No releases or tags found for {repo} and no images configured"
            )
            return []

        repo_results: list[RepositoryInfo] = []
        for image in images:
            component_name = f"{repo}/{image.split('/')[-1]}"
            logger.info(f"Checking {component_name} latest commit")

            try:
                commit_info = self.find_latest_commit_with_image(repo, image)
                if commit_info:
                    commit_sha, tag = commit_info
                    repo_results.append(
                        {
                            "repository": f"https://github.com/{repo}",
                            "ref": commit_sha,
                            "image_url": f"{image}:{tag}",
                        }
                    )
                else:
                    logger.warning(f"No images found for {component_name}")
            except Exception as e:
                logger.error(f"Error finding commit with image for {repo}/{image}: {e}")

        return repo_results

    @staticmethod
    def save_to_release_candidates(
        results: list[RepositoryInfo], filename: str = "release-candidates.yaml"
    ) -> None:
        repo = ReleaseCandidatesRepository(filename)
        candidate = ReleaseCandidate(components=results)
        if repo.save(candidate):
            logger.info(
                f"Saved new release candidate {candidate.metadata.get('id')} to {filename}"
            )
        else:
            logger.info("No changes detected or error occurred, skipping update")


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
