from enum import StrEnum

class ReferralReasonCountReason(StrEnum):
    ALREADY = "already"
    DISPOSABLE = "disposable"
    LEFT = "left"
    NEVER_CONFIRMED = "never_confirmed"
    REVIEWER = "reviewer"
    SAME_IP = "same_ip"
    SELF = "self"

    def __str__(self) -> str:
        return str(self.value)
