from enum import StrEnum

class SponsorRequestItemPaidSource(StrEnum):
    MANUAL = "manual"
    STRIPE = "stripe"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)
