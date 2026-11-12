from enum import StrEnum

class ConnectDoneRowStatus(StrEnum):
    DONE = "done"
    FAILED = "failed"
    PENDING = "pending"
    SKIPPED = "skipped"
    VERIFIED = "verified"

    def __str__(self) -> str:
        return str(self.value)
