from enum import StrEnum

class CouponPoolListItemSource(StrEnum):
    SHOPIFY = "shopify"
    UPLOAD = "upload"
    WOOCOMMERCE = "woocommerce"

    def __str__(self) -> str:
        return str(self.value)
