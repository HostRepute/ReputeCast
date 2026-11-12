from enum import StrEnum

class ReadinessRouteStream(StrEnum):
    AUTOMATION = "automation"
    BROADCAST = "broadcast"
    SYSTEM = "system"

    def __str__(self) -> str:
        return str(self.value)
