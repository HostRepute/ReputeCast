from enum import StrEnum

class PlacementCopyDetailReadVia(StrEnum):
    GMAIL = "gmail"
    GRAPH = "graph"
    IMAP = "imap"

    def __str__(self) -> str:
        return str(self.value)
