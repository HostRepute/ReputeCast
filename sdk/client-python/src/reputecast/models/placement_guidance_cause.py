from enum import StrEnum

class PlacementGuidanceCause(StrEnum):
    AUTH = "auth"
    READINESS = "readiness"
    REPUTATION = "reputation"
    WARMUP = "warmup"

    def __str__(self) -> str:
        return str(self.value)
