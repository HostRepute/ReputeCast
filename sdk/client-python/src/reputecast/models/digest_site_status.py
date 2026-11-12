from enum import StrEnum

class DigestSiteStatus(StrEnum):
    OK = "ok"
    OUTDATED = "outdated"
    SILENT = "silent"

    def __str__(self) -> str:
        return str(self.value)
