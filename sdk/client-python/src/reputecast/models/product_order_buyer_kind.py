from enum import StrEnum

class ProductOrderBuyerKind(StrEnum):
    CONTACT = "contact"
    JOINED = "joined"
    NONE = "none"

    def __str__(self) -> str:
        return str(self.value)
