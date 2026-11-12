from enum import StrEnum

class EarnImportSource(StrEnum):
    BEEHIIV = "beehiiv"
    CSV = "csv"
    GHOST = "ghost"
    KIT = "kit"
    SUBSTACK = "substack"

    def __str__(self) -> str:
        return str(self.value)
