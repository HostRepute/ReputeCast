from enum import StrEnum

class TrackerRecentEventKind(StrEnum):
    CART = "cart"
    CUSTOM = "custom"
    PAGE = "page"
    PRODUCT = "product"

    def __str__(self) -> str:
        return str(self.value)
