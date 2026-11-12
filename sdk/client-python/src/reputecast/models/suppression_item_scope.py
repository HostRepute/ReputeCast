from enum import StrEnum

class SuppressionItemScope(StrEnum):
    ALL = "all"
    MARKETING = "marketing"

    def __str__(self) -> str:
        return str(self.value)
