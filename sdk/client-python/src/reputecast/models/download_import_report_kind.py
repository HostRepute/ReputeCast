from enum import StrEnum

class DownloadImportReportKind(StrEnum):
    INVALID = "invalid"
    REPORT = "report"
    SKIPPED = "skipped"
    UNREADABLE = "unreadable"

    def __str__(self) -> str:
        return str(self.value)
