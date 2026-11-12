from enum import StrEnum

class PageDiffBlockKind(StrEnum):
    ADDED = "added"
    CHANGED = "changed"
    MOVED = "moved"
    REMOVED = "removed"

    def __str__(self) -> str:
        return str(self.value)
