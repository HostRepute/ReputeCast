from enum import StrEnum

class CommerceBlockSoldOut(StrEnum):
    HIDE = "hide"
    LABEL = "label"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)
