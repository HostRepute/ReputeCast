from enum import StrEnum

class ScoringBandMoveFrom(StrEnum):
    COLD = "cold"
    HOT = "hot"
    WARM = "warm"

    def __str__(self) -> str:
        return str(self.value)
