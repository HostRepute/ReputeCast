from enum import StrEnum

class ApprovalWebsiteSnapshotKind(StrEnum):
    CAMPAIGN = "campaign"
    WEB = "web"

    def __str__(self) -> str:
        return str(self.value)
