from enum import StrEnum

class ProductOrderItemDisputeStatus(StrEnum):
    LOST = "lost"
    OPEN = "open"
    VALUE_0 = ""
    WON = "won"

    def __str__(self) -> str:
        return str(self.value)
