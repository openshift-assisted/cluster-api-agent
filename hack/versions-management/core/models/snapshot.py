from pydantic.dataclasses import dataclass
from .snapshot_metadata import SnapshotMetadata
from .commit import Commit

@dataclass(frozen=True)
class Snapshot:
    metadata: SnapshotMetadata
    commits: list[Commit]
