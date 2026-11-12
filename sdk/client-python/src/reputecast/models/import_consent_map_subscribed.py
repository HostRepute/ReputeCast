from enum import StrEnum

class ImportConsentMapSubscribed(StrEnum):
    CONFIRMED = "confirmed"
    PENDING = "pending"
    SKIP = "skip"

    def __str__(self) -> str:
        return str(self.value)
