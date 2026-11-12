from enum import StrEnum

class ProviderAwaitingMessageStatus(StrEnum):
    ACCEPTED = "accepted"
    BOUNCED = "bounced"
    COMPLAINED = "complained"
    DEFERRED = "deferred"
    DELIVERED = "delivered"
    REJECTED = "rejected"

    def __str__(self) -> str:
        return str(self.value)
