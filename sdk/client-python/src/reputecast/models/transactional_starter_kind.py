from enum import StrEnum

class TransactionalStarterKind(StrEnum):
    ESSENTIAL = "essential"
    OPTIONAL = "optional"

    def __str__(self) -> str:
        return str(self.value)
