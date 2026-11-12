from enum import StrEnum

class ContactScoreBand(StrEnum):
    COLD = "cold"
    HOT = "hot"
    WARM = "warm"

    def __str__(self) -> str:
        return str(self.value)
