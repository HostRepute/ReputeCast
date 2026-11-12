from enum import StrEnum

class CommerceContactOrderCreditedKind(StrEnum):
    AUTOMATION = "automation"
    CAMPAIGN = "campaign"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)
