from enum import StrEnum

class BookInputStatus(StrEnum):
    BOOKED = "booked"
    HOLD = "hold"

    def __str__(self) -> str:
        return str(self.value)
