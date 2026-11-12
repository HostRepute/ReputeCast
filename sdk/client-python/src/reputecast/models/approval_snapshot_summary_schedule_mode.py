from enum import StrEnum

class ApprovalSnapshotSummaryScheduleMode(StrEnum):
    AT = "at"
    BEST_TIME = "best_time"
    LOCAL = "local"
    NOW = "now"

    def __str__(self) -> str:
        return str(self.value)
