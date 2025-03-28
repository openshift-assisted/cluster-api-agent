from pydantic.dataclasses import dataclass

@dataclass(frozen=True)
class Commit:
    repository: str
    ref: str
    image_url: str | None = None
