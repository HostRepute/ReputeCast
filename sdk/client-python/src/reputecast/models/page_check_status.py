from enum import StrEnum

class PageCheckStatus(StrEnum):
    BLOCK = "block"
    PASS = "pass"
    WARN = "warn"

    def __str__(self) -> str:
        return str(self.value)
