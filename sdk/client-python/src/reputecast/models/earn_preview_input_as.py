from enum import StrEnum

class EarnPreviewInputAs(StrEnum):
    FREE = "free"
    OFFER = "offer"
    PAID = "paid"
    VISITOR = "visitor"

    def __str__(self) -> str:
        return str(self.value)
