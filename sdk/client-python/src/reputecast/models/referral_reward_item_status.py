from enum import StrEnum

class ReferralRewardItemStatus(StrEnum):
    EXPIRED = "expired"
    FAILED = "failed"
    FULFILLED = "fulfilled"
    PENDING = "pending"
    SENT = "sent"
    TO_FULFILL = "to_fulfill"
    WAITING_CODES = "waiting_codes"

    def __str__(self) -> str:
        return str(self.value)
