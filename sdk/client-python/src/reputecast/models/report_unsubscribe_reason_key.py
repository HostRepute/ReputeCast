from enum import StrEnum

class ReportUnsubscribeReasonKey(StrEnum):
    NEVER_SIGNED_UP = "never_signed_up"
    NONE = "none"
    NOT_RELEVANT = "not_relevant"
    OTHER = "other"
    SPAM = "spam"
    TOO_FREQUENT = "too_frequent"

    def __str__(self) -> str:
        return str(self.value)
