from enum import StrEnum

class ApprovalWebsiteSnapshotWhen(StrEnum):
    AT = "at"
    NOW = "now"
    WITH_SEND = "with_send"

    def __str__(self) -> str:
        return str(self.value)
