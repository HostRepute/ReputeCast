from enum import StrEnum

class SuppressionSyncRowStatus(StrEnum):
    DONE = "done"
    ERROR = "error"
    IN_SYNC = "in_sync"
    LOCAL = "local"
    OFF = "off"
    SYNCING = "syncing"

    def __str__(self) -> str:
        return str(self.value)
