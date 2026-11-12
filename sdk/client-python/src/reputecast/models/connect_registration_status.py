from enum import StrEnum

class ConnectRegistrationStatus(StrEnum):
    DONE = "done"
    FAILED = "failed"
    MANUAL = "manual"
    SKIPPED = "skipped"

    def __str__(self) -> str:
        return str(self.value)
