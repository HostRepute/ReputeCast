from enum import StrEnum

class AIActivityRowOutcome(StrEnum):
    ACCEPTED = "accepted"
    ACCEPTED_EDITS = "accepted_edits"
    CANCELLED = "cancelled"
    DISCARDED = "discarded"
    ERROR = "error"
    KEPT = "kept"
    NOTHING_DRAFTED = "nothing_drafted"
    PENDING = "pending"
    REFUSED = "refused"
    REJECTED = "rejected"

    def __str__(self) -> str:
        return str(self.value)
