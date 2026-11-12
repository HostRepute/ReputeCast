from enum import StrEnum

class ApprovalPageSnapshotVisibility(StrEnum):
    MEMBERS = "members"
    PAID = "paid"
    PUBLIC = "public"

    def __str__(self) -> str:
        return str(self.value)
