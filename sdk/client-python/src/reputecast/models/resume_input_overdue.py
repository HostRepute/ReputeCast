from enum import StrEnum

class ResumeInputOverdue(StrEnum):
    SEND = "send"
    SKIP = "skip"
    SPREAD = "spread"
    VALUE_3 = ""

    def __str__(self) -> str:
        return str(self.value)
