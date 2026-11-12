from enum import StrEnum

class PlacementSeedViewStatus(StrEnum):
    CONNECTED = "connected"
    EXPIRED = "expired"
    FAILED = "failed"
    UNCHECKED = "unchecked"

    def __str__(self) -> str:
        return str(self.value)
