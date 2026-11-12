from enum import StrEnum

class VerifyContactInputService(StrEnum):
    BOUNCER = "bouncer"
    EMAILABLE = "emailable"
    KICKBOX = "kickbox"
    NEVERBOUNCE = "neverbounce"
    VALUE_5 = ""
    ZEROBOUNCE = "zerobounce"

    def __str__(self) -> str:
        return str(self.value)
