from enum import StrEnum

class ContactActivityFilter(StrEnum):
    ALL = "all"
    CONSENT = "consent"
    DELIVERIES = "deliveries"
    ENGAGEMENT = "engagement"

    def __str__(self) -> str:
        return str(self.value)
