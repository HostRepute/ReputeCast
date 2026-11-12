from enum import StrEnum

class ImportConsentMapTransactional(StrEnum):
    PENDING = "pending"
    SKIP = "skip"

    def __str__(self) -> str:
        return str(self.value)
