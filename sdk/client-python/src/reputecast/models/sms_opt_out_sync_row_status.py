from enum import StrEnum

class SmsOptOutSyncRowStatus(StrEnum):
    FAILING = "failing"
    IN_SYNC = "in_sync"

    def __str__(self) -> str:
        return str(self.value)
