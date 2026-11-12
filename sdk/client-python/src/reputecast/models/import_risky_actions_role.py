from enum import StrEnum

class ImportRiskyActionsRole(StrEnum):
    EXCLUDE_BROADCASTS = "exclude_broadcasts"
    IMPORT = "import"
    SKIP = "skip"

    def __str__(self) -> str:
        return str(self.value)
