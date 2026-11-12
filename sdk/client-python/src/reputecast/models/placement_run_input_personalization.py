from enum import StrEnum

class PlacementRunInputPersonalization(StrEnum):
    DATA = "data"
    FALLBACKS = "fallbacks"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)
