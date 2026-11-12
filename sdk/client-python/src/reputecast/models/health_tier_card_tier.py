from enum import StrEnum

class HealthTierCardTier(StrEnum):
    ACTIVE = "active"
    AT_RISK = "at_risk"
    COOLING = "cooling"
    SUNSET = "sunset"

    def __str__(self) -> str:
        return str(self.value)
