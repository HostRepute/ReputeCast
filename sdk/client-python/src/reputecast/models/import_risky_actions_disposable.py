from enum import StrEnum

class ImportRiskyActionsDisposable(StrEnum):
    IMPORT = "import"
    SKIP = "skip"

    def __str__(self) -> str:
        return str(self.value)
