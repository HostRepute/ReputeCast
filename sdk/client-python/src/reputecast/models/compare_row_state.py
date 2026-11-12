from enum import StrEnum

class CompareRowState(StrEnum):
    CHANGED = "changed"
    NEW = "new"
    ONLY = "only"
    REMOVED = "removed"
    STILL = "still"
    UP_TO_DATE = "up_to_date"

    def __str__(self) -> str:
        return str(self.value)
