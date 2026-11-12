from enum import StrEnum

class AutomationDiffItemKind(StrEnum):
    ADDED = "added"
    CHANGED = "changed"
    REMOVED = "removed"
    UNCHANGED = "unchanged"

    def __str__(self) -> str:
        return str(self.value)
