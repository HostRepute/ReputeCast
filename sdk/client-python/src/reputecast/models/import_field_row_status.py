from enum import StrEnum

class ImportFieldRowStatus(StrEnum):
    MAPPED = "mapped"
    NEW = "new"
    SKIPPED = "skipped"

    def __str__(self) -> str:
        return str(self.value)
