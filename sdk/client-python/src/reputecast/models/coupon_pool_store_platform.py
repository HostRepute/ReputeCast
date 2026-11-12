from enum import StrEnum

class CouponPoolStorePlatform(StrEnum):
    SHOPIFY = "shopify"
    WOOCOMMERCE = "woocommerce"

    def __str__(self) -> str:
        return str(self.value)
