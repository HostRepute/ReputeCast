from enum import StrEnum

class ImportRiskyActionsTrap(StrEnum):
    IMPORT = "import"
    SKIP = "skip"

    def __str__(self) -> str:
        return str(self.value)
