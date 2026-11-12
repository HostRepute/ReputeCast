from enum import StrEnum

class ReadinessFixKind(StrEnum):
    APPLY = "apply"
    LINK = "link"

    def __str__(self) -> str:
        return str(self.value)
