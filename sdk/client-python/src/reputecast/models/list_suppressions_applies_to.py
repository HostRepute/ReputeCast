from enum import StrEnum

class ListSuppressionsAppliesTo(StrEnum):
    ALL = "all"
    MARKETING = "marketing"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)
