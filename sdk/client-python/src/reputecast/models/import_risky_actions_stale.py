from enum import StrEnum

class ImportRiskyActionsStale(StrEnum):
    IMPORT = "import"
    PENDING = "pending"
    SKIP = "skip"

    def __str__(self) -> str:
        return str(self.value)
