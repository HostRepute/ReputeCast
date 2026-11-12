from enum import StrEnum

class ReadinessRouteShareRateFrom(StrEnum):
    DEFAULT = "default"
    PUBLISHED = "published"
    WARMUP = "warmup"

    def __str__(self) -> str:
        return str(self.value)
