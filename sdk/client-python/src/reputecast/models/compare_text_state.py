from enum import StrEnum

class CompareTextState(StrEnum):
    CHANGED = "changed"
    UP_TO_DATE = "up_to_date"

    def __str__(self) -> str:
        return str(self.value)
