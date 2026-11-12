from enum import StrEnum

class ListHealthRowKind(StrEnum):
    LIST = "list"
    VIEW = "view"

    def __str__(self) -> str:
        return str(self.value)
