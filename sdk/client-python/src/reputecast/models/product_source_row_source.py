from enum import StrEnum

class ProductSourceRowSource(StrEnum):
    PAGE = "page"
    PRODUCT_PAGE = "product_page"
    SHOP = "shop"

    def __str__(self) -> str:
        return str(self.value)
