from enum import StrEnum

class LeadAdsReceivingReason(StrEnum):
    NOT_SUBSCRIBED = "not_subscribed"
    QUIET = "quiet"
    UNREACHABLE = "unreachable"
    VALUE_3 = ""

    def __str__(self) -> str:
        return str(self.value)
