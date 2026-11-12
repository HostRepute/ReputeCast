from enum import StrEnum

class SuppressionProviderSyncStatus(StrEnum):
    ERROR = "error"
    IN_SYNC = "in_sync"
    OFF = "off"
    SYNCING = "syncing"

    def __str__(self) -> str:
        return str(self.value)
