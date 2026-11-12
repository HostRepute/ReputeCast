from enum import StrEnum

class ReadinessAudienceRowKind(StrEnum):
    BASE = "base"
    MINUS = "minus"
    PLUS = "plus"

    def __str__(self) -> str:
        return str(self.value)
