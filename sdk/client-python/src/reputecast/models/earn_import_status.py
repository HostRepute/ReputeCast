from enum import StrEnum

class EarnImportStatus(StrEnum):
    CANCELED = "canceled"
    DONE = "done"
    FAILED = "failed"
    IMPORTING = "importing"
    MATCHED = "matched"
    UPLOADED = "uploaded"

    def __str__(self) -> str:
        return str(self.value)
