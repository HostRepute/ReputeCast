from enum import StrEnum

class PushSiteUpdateWorkerMode(StrEnum):
    FILE = "file"
    IMPORT = "import"

    def __str__(self) -> str:
        return str(self.value)
