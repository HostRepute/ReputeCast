from enum import StrEnum

class OfferDetailKind(StrEnum):
    REFERRAL = "referral"
    RETENTION = "retention"
    SIGNUP = "signup"

    def __str__(self) -> str:
        return str(self.value)
