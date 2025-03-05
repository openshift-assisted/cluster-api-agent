#!/usr/bin/env python3

import logging
import os
import sys
import uuid
from datetime import datetime

from ruamel.yaml import YAML

script_dir = os.path.dirname(os.path.abspath(__file__))
project_root = os.path.dirname(script_dir)
if project_root not in sys.path:
    sys.path.insert(0, project_root)

from hack.shared_types import SnapshotCollection, RepositoryInfo, Snapshot

logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger("release-repository")

# Configure YAML parser
yaml = YAML(typ="rt")
yaml.default_flow_style = False
yaml.explicit_start = False
yaml.preserve_quotes = True
yaml.width = 4096
yaml.indent(mapping=2, sequence=2, offset=0)


class ReleaseCandidate:

    def __init__(
        self,
        components: list[RepositoryInfo],
        generated_at: str | None = None,
        status: str = "pending",
        tested_at: str | None = None,
        id: str | None = None,
    ):
        self.metadata = {
            "generated_at": generated_at or datetime.now().isoformat(),
            "status": status,
        }

        if tested_at:
            self.metadata["tested_at"] = tested_at

        if id:
            self.metadata["id"] = id
        else:
            self.metadata["id"] = str(uuid.uuid4())

        self.components = components

    def to_dict(self) -> Snapshot:
        return {"metadata": self.metadata, "components": self.components}

    @classmethod
    def from_dict(cls, data: Snapshot) -> "ReleaseCandidate":
        metadata = data.get("metadata", {})
        components = data.get("components", [])

        return cls(
            components=components,
            generated_at=metadata.get("generated_at"),
            status=metadata.get("status", "pending"),
            tested_at=metadata.get("tested_at"),
            id=metadata.get("id"),
        )

    def equals(self, other: "ReleaseCandidate") -> bool:
        set1 = {
            (d.get("repository", ""), d.get("ref", ""), d.get("image_url", ""))
            for d in self.components
        }
        set2 = {
            (d.get("repository", ""), d.get("ref", ""), d.get("image_url", ""))
            for d in other.components
        }
        return set1 == set2


class ReleaseCandidatesRepository:

    def __init__(self, file_path: str = "release-candidates.yaml"):
        self.file_path = file_path

    def as_list(self) -> list[ReleaseCandidate]:
        data = self._read_file()
        snapshots = data.get("snapshots", [])

        return [ReleaseCandidate.from_dict(snapshot) for snapshot in snapshots]

    def find_by_status(self, status: str) -> list[ReleaseCandidate]:
        candidates = self.as_list()
        return [rc for rc in candidates if rc.metadata.get("status") == status]

    def find_by_id(self, id: str) -> ReleaseCandidate | None:
        candidates = self.as_list()
        for candidate in candidates:
            if candidate.metadata.get("id") == id:
                return candidate
        return None

    def save(self, candidate: ReleaseCandidate) -> bool:
        data = self._read_file()

        candidate_dict = candidate.to_dict()

        candidates = data.get("snapshots", [])
        for i, existing in enumerate(candidates):
            existing_id = existing.get("metadata", {}).get("id")
            if existing_id and existing_id == candidate.metadata.get("id"):
                candidates[i] = candidate_dict
                data["snapshots"] = candidates
                self._write_file(data)
                return True

        if "snapshots" not in data:
            data["snapshots"] = []

        new_rc = ReleaseCandidate.from_dict(candidate_dict)
        for existing in self.as_list():
            if new_rc.equals(existing):
                logger.info("Similar release candidate already exists, skipping")
                return False

        data["snapshots"].insert(0, candidate_dict)
        self._write_file(data)
        return True

    def update_status(self, id: str, status: str, tested_at: str = None) -> bool:
        candidate = self.find_by_id(id)
        if not candidate:
            return False

        candidate.metadata["status"] = status
        if tested_at or status in ["successful", "failed"]:
            candidate.metadata["tested_at"] = tested_at

        return self.save(candidate)

    def _read_file(self) -> SnapshotCollection:
        if not os.path.exists(self.file_path):
            return {"snapshots": []}

        try:
            with open(self.file_path, "r") as f:
                data = yaml.load(f)
                return data or {"snapshots": []}
        except Exception as e:
            logger.error(f"Error reading {self.file_path}: {e}")
            return {"snapshots": []}

    def _write_file(self, data: SnapshotCollection) -> None:
        try:
            with open(self.file_path, "w") as f:
                yaml.dump(data, f)
        except Exception as e:
            logger.error(f"Error writing to {self.file_path}: {e}")
            raise
