from enum import StrEnum

class CouponPoolSettingsKind(StrEnum):
    FIXED_AMOUNT = "fixed_amount"
    FREE_SHIPPING = "free_shipping"
    PERCENTAGE = "percentage"
    VALUE_3 = ""

    def __str__(self) -> str:
        return str(self.value)
