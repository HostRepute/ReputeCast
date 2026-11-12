from enum import StrEnum

class ListProductOrdersStatus(StrEnum):
    DISPUTED = "disputed"
    PAID = "paid"
    PARTIALLY_REFUNDED = "partially_refunded"
    REFUNDED = "refunded"
    TEST = "test"
    VALUE_0 = ""

    def __str__(self) -> str:
        return str(self.value)
