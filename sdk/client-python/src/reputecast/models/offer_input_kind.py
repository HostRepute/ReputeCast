from enum import StrEnum

class OfferInputKind(StrEnum):
    REFERRAL = "referral"
    RETENTION = "retention"
    SIGNUP = "signup"

    def __str__(self) -> str:
        return str(self.value)
