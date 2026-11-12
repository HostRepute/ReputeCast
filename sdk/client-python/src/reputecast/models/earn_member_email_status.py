from enum import StrEnum

class EarnMemberEmailStatus(StrEnum):
    DRAFT = "draft"
    LIVE = "live"
    MISSING = "missing"

    def __str__(self) -> str:
        return str(self.value)
