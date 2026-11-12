from enum import StrEnum

class OfferKind(StrEnum):
    REFERRAL = "referral"
    RETENTION = "retention"
    SIGNUP = "signup"

    def __str__(self) -> str:
        return str(self.value)
