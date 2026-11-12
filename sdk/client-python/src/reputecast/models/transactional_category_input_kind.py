from enum import StrEnum

class TransactionalCategoryInputKind(StrEnum):
    ESSENTIAL = "essential"
    OPTIONAL = "optional"

    def __str__(self) -> str:
        return str(self.value)
