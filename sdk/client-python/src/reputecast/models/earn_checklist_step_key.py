from enum import StrEnum

class EarnChecklistStepKey(StrEnum):
    CONNECT = "connect"
    EMAILS = "emails"
    LIVE = "live"
    PAGE = "page"
    TEST = "test"
    TIER = "tier"

    def __str__(self) -> str:
        return str(self.value)
