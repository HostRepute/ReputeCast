from enum import StrEnum

class ImportConsentMapPending(StrEnum):
    PENDING = "pending"
    SKIP = "skip"

    def __str__(self) -> str:
        return str(self.value)
