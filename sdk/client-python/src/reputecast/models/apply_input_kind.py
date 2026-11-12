from enum import StrEnum

class ApplyInputKind(StrEnum):
    IMAGE = "image"
    LINK = "link"

    def __str__(self) -> str:
        return str(self.value)
