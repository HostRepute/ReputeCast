from enum import StrEnum

class PersonalizationTokenKind(StrEnum):
    CONTACT = "contact"
    COUPON = "coupon"
    CUSTOM = "custom"
    PAID = "paid"
    REFERRAL = "referral"
    SYSTEM = "system"

    def __str__(self) -> str:
        return str(self.value)
