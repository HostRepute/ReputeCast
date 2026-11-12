from enum import StrEnum

class ContactListJoinStatus(StrEnum):
    PENDING = "pending"
    SUBSCRIBED = "subscribed"

    def __str__(self) -> str:
        return str(self.value)
