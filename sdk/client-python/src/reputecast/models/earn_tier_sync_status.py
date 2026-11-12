from enum import StrEnum

class EarnTierSyncStatus(StrEnum):
    DRAFT = "draft"
    FAILED = "failed"
    SYNCED = "synced"

    def __str__(self) -> str:
        return str(self.value)
