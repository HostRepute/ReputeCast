from enum import StrEnum

class TransactionalDiffLineKind(StrEnum):
    ADD = "add"
    CTX = "ctx"
    DEL = "del"

    def __str__(self) -> str:
        return str(self.value)
