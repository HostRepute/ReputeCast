from enum import StrEnum

class ListContactsTier(StrEnum):
    ACTIVE = "active"
    AT_RISK = "at_risk"
    COOLING = "cooling"
    SUNSET = "sunset"
    VALUE_4 = ""

    def __str__(self) -> str:
        return str(self.value)
