from enum import StrEnum

class ListSuppressionsReason(StrEnum):
    COMPLAINT = "complaint"
    HARD_BOUNCE = "hard_bounce"
    INVALID = "invalid"
    MANUAL = "manual"
    SPAM_TRAP = "spam_trap"
    UNSUBSCRIBE = "unsubscribe"
    VALUE_6 = ""

    def __str__(self) -> str:
        return str(self.value)
