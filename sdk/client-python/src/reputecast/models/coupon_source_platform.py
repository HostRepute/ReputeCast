from enum import StrEnum

class CouponSourcePlatform(StrEnum):
    SHOPIFY = "shopify"
    WOOCOMMERCE = "woocommerce"

    def __str__(self) -> str:
        return str(self.value)
