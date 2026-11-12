from enum import StrEnum

class DigestThresholdsInputScope(StrEnum):
    CAMPAIGN = "campaign"
    ROUTE = "route"

    def __str__(self) -> str:
        return str(self.value)
