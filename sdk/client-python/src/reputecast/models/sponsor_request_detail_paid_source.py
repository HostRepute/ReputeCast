from enum import StrEnum

class SponsorRequestDetailPaidSource(StrEnum):
    MANUAL = "manual"
    STRIPE = "stripe"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)
