from enum import StrEnum

class ImportRiskyActionsTypo(StrEnum):
    FIX = "fix"
    IMPORT = "import"
    SKIP = "skip"

    def __str__(self) -> str:
        return str(self.value)
