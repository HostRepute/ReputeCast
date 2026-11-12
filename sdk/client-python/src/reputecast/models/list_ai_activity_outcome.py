from enum import StrEnum

class ListAiActivityOutcome(StrEnum):
    ACCEPTED = "accepted"
    ERROR = "error"
    KEPT = "kept"
    NOTHING_DRAFTED = "nothing_drafted"
    PENDING = "pending"
    REFUSED = "refused"
    REJECTED = "rejected"

    def __str__(self) -> str:
        return str(self.value)
