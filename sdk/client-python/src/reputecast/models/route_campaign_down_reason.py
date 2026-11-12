from enum import StrEnum

class RouteCampaignDownReason(StrEnum):
    CHECKS = "checks"
    NO_FALLBACK = "no_fallback"
    PAUSE_AND_ASK = "pause_and_ask"
    WARMING = "warming"

    def __str__(self) -> str:
        return str(self.value)
