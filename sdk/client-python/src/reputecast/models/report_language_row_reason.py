from enum import StrEnum

class ReportLanguageRowReason(StrEnum):
    DRAFT = "draft"
    HELD = "held"
    MISSING = "missing"

    def __str__(self) -> str:
        return str(self.value)
