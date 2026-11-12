from enum import StrEnum

class ProductOrderItemStatus(StrEnum):
    DISPUTED = "disputed"
    PAID = "paid"
    PARTIALLY_REFUNDED = "partially_refunded"
    REFUNDED = "refunded"

    def __str__(self) -> str:
        return str(self.value)
