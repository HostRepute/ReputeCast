from enum import StrEnum

class DigestRenderInputPreview(StrEnum):
    LAST = "last"
    NEXT = "next"
    ONE = "one"
    SIX = "six"

    def __str__(self) -> str:
        return str(self.value)
