#!/usr/bin/env python3

from typing import TypedDict

class RepositoryInfo(TypedDict):
    repository: str
    ref: str
    image_url: str | None

class SnapshotMetadata(TypedDict):
    generated_at: str
    status: str
    tested_at: str

class SnapshotInfo(TypedDict):
    metadata: SnapshotMetadata
    components: list[RepositoryInfo]

class ReleaseCandidates(TypedDict):
    snapshots: list[SnapshotInfo]

class VersionEntry(TypedDict):
    name: str
    components: list[RepositoryInfo]

class VersionsFile(TypedDict):
    components: list[VersionEntry]

class RepositoryConfig(TypedDict, total=False):
    images: list[str] | None
    version_prefix: str
