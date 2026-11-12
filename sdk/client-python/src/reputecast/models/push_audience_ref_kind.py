from enum import StrEnum

class PushAudienceRefKind(StrEnum):
    LIST = "list"
    SEGMENT = "segment"

    def __str__(self) -> str:
        return str(self.value)
