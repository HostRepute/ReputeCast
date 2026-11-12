from enum import StrEnum

class TransactionalDraftSaveSource(StrEnum):
    AUTOSAVE = "autosave"
    IMPORT = "import"
    MANUAL = "manual"
    RESTORE = "restore"

    def __str__(self) -> str:
        return str(self.value)
