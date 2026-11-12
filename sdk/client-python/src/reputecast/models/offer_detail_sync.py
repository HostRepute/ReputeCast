from enum import StrEnum

class OfferDetailSync(StrEnum):
    DRAFT = "draft"
    FAILED = "failed"
    SYNCED = "synced"

    def __str__(self) -> str:
        return str(self.value)
