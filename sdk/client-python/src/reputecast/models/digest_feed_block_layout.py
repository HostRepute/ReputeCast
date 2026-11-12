from enum import StrEnum

class DigestFeedBlockLayout(StrEnum):
    CARDS = "cards"
    GRID = "grid"
    LIST = "list"

    def __str__(self) -> str:
        return str(self.value)
