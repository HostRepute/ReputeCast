from enum import StrEnum

class ReferralTimelineEventKind(StrEnum):
    COUNTED = "counted"
    JOINED = "joined"
    MILESTONE = "milestone"
    PARTNER_REQUEST = "partner_request"
    REWARD_SENT = "reward_sent"

    def __str__(self) -> str:
        return str(self.value)
