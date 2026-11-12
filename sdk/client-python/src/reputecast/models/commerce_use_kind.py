from enum import StrEnum

class CommerceUseKind(StrEnum):
    AUTOMATION = "automation"
    CAMPAIGN = "campaign"
    REPORTS = "reports"

    def __str__(self) -> str:
        return str(self.value)
