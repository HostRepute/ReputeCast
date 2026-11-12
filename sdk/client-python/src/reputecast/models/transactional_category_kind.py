from enum import StrEnum

class TransactionalCategoryKind(StrEnum):
    ESSENTIAL = "essential"
    OPTIONAL = "optional"

    def __str__(self) -> str:
        return str(self.value)
