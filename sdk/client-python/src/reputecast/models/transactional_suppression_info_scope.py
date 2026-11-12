from enum import StrEnum

class TransactionalSuppressionInfoScope(StrEnum):
    ALL = "all"
    MARKETING = "marketing"

    def __str__(self) -> str:
        return str(self.value)
