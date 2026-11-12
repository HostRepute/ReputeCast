from enum import StrEnum

class BlockOptionKind(StrEnum):
    LINK = "link"
    PARTNER = "partner"

    def __str__(self) -> str:
        return str(self.value)
