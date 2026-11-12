from enum import StrEnum

class ListContactsBand(StrEnum):
    COLD = "cold"
    HOT = "hot"
    NONE = "none"
    VALUE_4 = ""
    WARM = "warm"

    def __str__(self) -> str:
        return str(self.value)
