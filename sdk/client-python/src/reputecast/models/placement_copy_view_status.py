from enum import StrEnum

class PlacementCopyViewStatus(StrEnum):
    FAILED = "failed"
    FOUND = "found"
    NOT_ARRIVED = "not_arrived"
    NOT_CHECKED = "not_checked"
    QUEUED = "queued"
    SENT = "sent"

    def __str__(self) -> str:
        return str(self.value)
