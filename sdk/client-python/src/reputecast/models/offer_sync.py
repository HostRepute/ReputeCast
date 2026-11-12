from enum import StrEnum

class OfferSync(StrEnum):
    DRAFT = "draft"
    FAILED = "failed"
    SYNCED = "synced"

    def __str__(self) -> str:
        return str(self.value)
