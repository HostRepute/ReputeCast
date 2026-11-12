from enum import StrEnum

class GiftEmailStatus(StrEnum):
    DRAFT = "draft"
    LIVE = "live"
    MISSING = "missing"

    def __str__(self) -> str:
        return str(self.value)
