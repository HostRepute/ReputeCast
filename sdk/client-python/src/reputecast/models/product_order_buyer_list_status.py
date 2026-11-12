from enum import StrEnum

class ProductOrderBuyerListStatus(StrEnum):
    PENDING = "pending"
    SUBSCRIBED = "subscribed"
    UNSUBSCRIBED = "unsubscribed"
    VALUE_0 = ""

    def __str__(self) -> str:
        return str(self.value)
