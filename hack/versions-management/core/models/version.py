from pydantic.dataclasses import dataclass
from .commit import Commit

@dataclass(frozen=True)
class Version:
    name: str
    commits: list[Commit]
