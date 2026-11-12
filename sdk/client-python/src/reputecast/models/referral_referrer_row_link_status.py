from enum import StrEnum

class ReferralReferrerRowLinkStatus(StrEnum):
    ACTIVE = "active"
    PAUSED = "paused"

    def __str__(self) -> str:
        return str(self.value)
