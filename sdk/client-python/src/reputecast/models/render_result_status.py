from enum import StrEnum

class RenderResultStatus(StrEnum):
    DONE = "done"
    FAILED = "failed"
    PENDING = "pending"

    def __str__(self) -> str:
        return str(self.value)
