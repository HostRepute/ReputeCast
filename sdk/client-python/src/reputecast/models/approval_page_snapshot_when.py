from enum import StrEnum

class ApprovalPageSnapshotWhen(StrEnum):
    AT = "at"
    NOW = "now"

    def __str__(self) -> str:
        return str(self.value)
