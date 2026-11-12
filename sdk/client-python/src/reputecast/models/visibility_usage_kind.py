from enum import StrEnum

class VisibilityUsageKind(StrEnum):
    FIELD = "field"
    LIST = "list"
    SEGMENT = "segment"
    TAG = "tag"

    def __str__(self) -> str:
        return str(self.value)
