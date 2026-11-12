from enum import StrEnum

class ImportRiskyActionsUnknown(StrEnum):
    IMPORT = "import"
    SKIP = "skip"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)
