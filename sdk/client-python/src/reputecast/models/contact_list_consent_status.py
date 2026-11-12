from enum import StrEnum

class ContactListConsentStatus(StrEnum):
    NONE = "none"
    PENDING = "pending"
    SUBSCRIBED = "subscribed"
    UNSUBSCRIBED = "unsubscribed"

    def __str__(self) -> str:
        return str(self.value)
