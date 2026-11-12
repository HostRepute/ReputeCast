from enum import StrEnum

class CommerceContactStoreMarketing(StrEnum):
    NONE = "none"
    NOT_SUBSCRIBED = "not_subscribed"
    PENDING = "pending"
    SUBSCRIBED = "subscribed"
    UNSUBSCRIBED = "unsubscribed"

    def __str__(self) -> str:
        return str(self.value)
