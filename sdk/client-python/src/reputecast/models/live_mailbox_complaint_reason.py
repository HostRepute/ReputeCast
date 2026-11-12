from enum import StrEnum

class LiveMailboxComplaintReason(StrEnum):
    NOT_REPORTED = "not_reported"

    def __str__(self) -> str:
        return str(self.value)
