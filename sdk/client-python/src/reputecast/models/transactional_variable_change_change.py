from enum import StrEnum

class TransactionalVariableChangeChange(StrEnum):
    ADDED = "added"
    CHANGED = "changed"
    REMOVED = "removed"

    def __str__(self) -> str:
        return str(self.value)
