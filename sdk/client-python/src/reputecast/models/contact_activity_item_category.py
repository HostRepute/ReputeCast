from enum import StrEnum

class ContactActivityItemCategory(StrEnum):
    CONSENT = "consent"
    DELIVERY = "delivery"
    ENGAGEMENT = "engagement"

    def __str__(self) -> str:
        return str(self.value)
