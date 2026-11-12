from enum import StrEnum

class ScoringUseKind(StrEnum):
    AUTOMATION = "automation"
    BLOCK = "block"
    SEGMENT = "segment"

    def __str__(self) -> str:
        return str(self.value)
