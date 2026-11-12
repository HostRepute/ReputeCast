from enum import StrEnum

class BookingInputStatus(StrEnum):
    BOOKED = "booked"
    HOLD = "hold"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)
