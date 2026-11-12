from enum import StrEnum

class PushWorkerMode(StrEnum):
    FILE = "file"
    IMPORT = "import"

    def __str__(self) -> str:
        return str(self.value)
