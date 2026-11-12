from enum import StrEnum

class LeadAdsActivityItemOutcome(StrEnum):
    BLOCKED = "blocked"
    CREATED = "created"
    DECLINED = "declined"
    INVALID = "invalid"
    PENDING = "pending"
    TEST = "test"
    UPDATED = "updated"

    def __str__(self) -> str:
        return str(self.value)
