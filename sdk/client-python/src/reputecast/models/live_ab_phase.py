from enum import StrEnum

class LiveABPhase(StrEnum):
    DECIDED = "decided"
    TESTING = "testing"

    def __str__(self) -> str:
        return str(self.value)
