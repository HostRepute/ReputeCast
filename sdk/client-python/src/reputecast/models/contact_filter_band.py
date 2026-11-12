from enum import StrEnum

class ContactFilterBand(StrEnum):
    COLD = "cold"
    HOT = "hot"
    NONE = "none"
    WARM = "warm"

    def __str__(self) -> str:
        return str(self.value)
