from enum import StrEnum

class TransactionalLanguageChangeChange(StrEnum):
    ADDED = "added"
    CHANGED = "changed"
    REMOVED = "removed"

    def __str__(self) -> str:
        return str(self.value)
