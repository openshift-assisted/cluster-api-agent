import hashlib
import logging
from datetime import datetime
from typing import override
from concurrent.futures import ThreadPoolExecutor

from core.clients.github_client import GitHubClient
from core.clients.image_registry_client import ImageRegistryClient
from core.models import Snapshot, SnapshotMetadata, Commit, Component
from core.repositories import ReleaseCandidateRepository
from core.repositories.components_repository import ComponentRepository
from core.services.service import Service
from core.utils.logging import setup_logger

class VersionDiscoveryService(Service):
    def __init__(self, rc_file_path: str, components_file_path: str):
        self.github: GitHubClient = GitHubClient()
        self.registry: ImageRegistryClient = ImageRegistryClient()
        self.rc_repository: ReleaseCandidateRepository = ReleaseCandidateRepository(rc_file_path)
        self.components_repository: ComponentRepository = ComponentRepository(components_file_path)
        self.logger: logging.Logger = setup_logger("VersionDiscoveryService")

    @override
    def run(self) -> None:
        commits: list[Commit] = []
        with ThreadPoolExecutor(max_workers=8) as executor:
            components = self.components_repository.find_all()
            for future in [executor.submit(self.process_repository, c) for c in components]:
                try:
                    result = future.result()
                    if result:
                        commits.append(result)
                except Exception as e:
                    raise Exception(f"Failed to resolve component: {e}") from e


        if not commits:
            raise Exception("No components discovered. Exiting.")

        snapshot_id = str(self._generate_components_hash(commits))
        snapshot = Snapshot(
            metadata=SnapshotMetadata(
                id=snapshot_id,
                generated_at=datetime.now(),
                status="pending",
            ),
            commits=commits,
        )

        if self.rc_repository.save(snapshot):
            self.logger.info(f"Snapshot {snapshot.metadata.id} has been saved successfully.")
        else:
            error_msg = f"Failed to save snapshot {snapshot.metadata.id}"
            self.logger.error(error_msg)
            raise Exception(error_msg)

    def process_repository(
        self, component: Component
    ) -> Commit:
        img_pattern = component.image_pattern
        repo = component.repository.removeprefix("https://github.com/")
        self.logger.info(f"Scanning repository {repo}")
        try:
            gh_repo = self.github.get_repo(repo)
            if component.versioning_selection_mechanism == "release":
                self.logger.info(f"Checking releases of component {component.name}")
                releases = [r for r in gh_repo.get_releases()[:20] if not r.prerelease]
                if releases:
                    latest = releases[0]
                    self.logger.info(f"Found release {latest} for repository {repo}")
                    return Commit(repository=f"https://github.com/{repo}", ref=latest.tag_name)

            elif component.versioning_selection_mechanism == "commit":
                self.logger.info(f"Checking commits of component {component.name}")
                for commit in gh_repo.get_commits()[:20]:
                    sha = commit.sha
                    tag = f"latest-{sha}"
                    img = f"{img_pattern}"
                    if self.registry.exists(img, tag):
                        self.logger.info(f"Found commit {sha} for repository {repo}")
                        return Commit(
                            repository=f"https://github.com/{repo}",
                            ref=sha,
                            image_url=f"{img}:{tag}",
                        )
            else:
                raise Exception(f"Versioning mechanism of component {component.repository} is not supported")
        except Exception as e:
            raise Exception(f"Failed to process {repo}: {e}") from e

    # using hash to create a reproducible id
    def _generate_components_hash(self, components: list[Commit]) -> str:
        sorted_components = sorted(components, key=lambda c: c.repository)
        component_str = ";".join([
            f"{c.repository}:{c.ref}:{c.image_url or ''}" for c in sorted_components
        ])
        return hashlib.md5(component_str.encode()).hexdigest()

