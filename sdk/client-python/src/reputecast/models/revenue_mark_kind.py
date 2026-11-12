from enum import StrEnum

class RevenueMarkKind(StrEnum):
    MOVED = "moved"
    OFFER = "offer"

    def __str__(self) -> str:
        return str(self.value)
