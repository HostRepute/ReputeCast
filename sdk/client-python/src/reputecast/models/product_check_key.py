from enum import StrEnum

class ProductCheckKey(StrEnum):
    COVER_ALT = "cover_alt"
    FAILED_NOTICE = "failed_notice"
    FILE = "file"
    PRICE = "price"
    ROUTE = "route"
    SALES_PAGE = "sales_page"
    STRIPE = "stripe"
    TEMPLATE = "template"

    def __str__(self) -> str:
        return str(self.value)
