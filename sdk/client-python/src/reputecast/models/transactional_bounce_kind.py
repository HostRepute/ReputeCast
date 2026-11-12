from enum import StrEnum

class TransactionalBounceKind(StrEnum):
    HARD = "hard"
    SOFT = "soft"

    def __str__(self) -> str:
        return str(self.value)
