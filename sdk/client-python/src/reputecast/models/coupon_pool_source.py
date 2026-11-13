from enum import StrEnum

class CouponPoolSource(StrEnum):
    SHOPIFY = "shopify"
    UPLOAD = "upload"
    WOOCOMMERCE = "woocommerce"

    def __str__(self) -> str:
        return str(self.value)
