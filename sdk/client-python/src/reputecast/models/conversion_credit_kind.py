from enum import StrEnum

class ConversionCreditKind(StrEnum):
    AUTOMATION = "automation"
    CAMPAIGN = "campaign"

    def __str__(self) -> str:
        return str(self.value)
