from enum import StrEnum

class ContactFilterTier(StrEnum):
    ACTIVE = "active"
    AT_RISK = "at_risk"
    COOLING = "cooling"
    SUNSET = "sunset"

    def __str__(self) -> str:
        return str(self.value)
