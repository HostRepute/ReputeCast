from enum import StrEnum

class ComposerVersionSource(StrEnum):
    API = "api"
    AUTOSAVE = "autosave"
    CONFLICT = "conflict"
    CREATE = "create"
    DUPLICATE = "duplicate"
    IMPORT = "import"
    MANUAL = "manual"
    MERGE = "merge"
    RECOVERED = "recovered"
    RESTORE = "restore"
    SYNCED = "synced"
    TEMPLATE = "template"

    def __str__(self) -> str:
        return str(self.value)
