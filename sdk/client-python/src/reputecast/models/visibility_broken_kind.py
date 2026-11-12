from enum import StrEnum

class VisibilityBrokenKind(StrEnum):
    FIELD = "field"
    LIST = "list"
    SEGMENT = "segment"

    def __str__(self) -> str:
        return str(self.value)
