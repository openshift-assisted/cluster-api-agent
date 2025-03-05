#!/usr/bin/env python3

from typing import TypedDict

class RepositoryInfo(TypedDict):
    repository: str
    ref: str
    image_url: str | None

class SnapshotMetadata(TypedDict):
    id: str
    generated_at: str
    status: str # "pending", "failed", "successful"
    tested_at: str

class Snapshot(TypedDict):
    metadata: SnapshotMetadata
    components: list[RepositoryInfo]

class SnapshotCollection(TypedDict):
    snapshots: list[Snapshot]

# a version is a promoted successful snapshot
class Version(TypedDict):
    name: str
    components: list[RepositoryInfo]

class VersionsCollection(TypedDict):
    versions: list[Version]

class RepositoryConfig(TypedDict, total=False):
    images: list[str] | None
    version_prefix: str
