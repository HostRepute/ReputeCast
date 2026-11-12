from enum import StrEnum

class ContactScoreUseKind(StrEnum):
    AUTOMATION = "automation"
    BLOCK = "block"
    SEGMENT = "segment"

    def __str__(self) -> str:
        return str(self.value)
