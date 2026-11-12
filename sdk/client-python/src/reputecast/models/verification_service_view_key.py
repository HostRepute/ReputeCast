from enum import StrEnum

class VerificationServiceViewKey(StrEnum):
    BOUNCER = "bouncer"
    EMAILABLE = "emailable"
    KICKBOX = "kickbox"
    NEVERBOUNCE = "neverbounce"
    ZEROBOUNCE = "zerobounce"

    def __str__(self) -> str:
        return str(self.value)
