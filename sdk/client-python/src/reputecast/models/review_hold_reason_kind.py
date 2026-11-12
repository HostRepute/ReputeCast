from enum import StrEnum

class ReviewHoldReasonKind(StrEnum):
    AUDIENCE_CHANGE = "audience_change"
    BLOCKED = "blocked"
    EDITED = "edited"
    NEW_WARNING = "new_warning"
    UNKNOWN = "unknown"

    def __str__(self) -> str:
        return str(self.value)
