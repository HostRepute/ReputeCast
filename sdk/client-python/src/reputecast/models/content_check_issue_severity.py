from enum import StrEnum

class ContentCheckIssueSeverity(StrEnum):
    BLOCK = "block"
    INFO = "info"
    PASS = "pass"
    WARN = "warn"

    def __str__(self) -> str:
        return str(self.value)
