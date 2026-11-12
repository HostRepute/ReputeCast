from enum import StrEnum

class WarmupRowReason(StrEnum):
    GUARD = "guard"
    HELD = "held"
    SHORT = "short"

    def __str__(self) -> str:
        return str(self.value)
