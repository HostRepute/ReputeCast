from enum import StrEnum

class AccessibilityImpactCampaignStatus(StrEnum):
    APPROVAL = "approval"
    APPROVED = "approved"
    DRAFT = "draft"
    SCHEDULED = "scheduled"

    def __str__(self) -> str:
        return str(self.value)
