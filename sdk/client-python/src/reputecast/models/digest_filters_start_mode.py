from enum import StrEnum

class DigestFiltersStartMode(StrEnum):
    AFTER_ON = "after_on"
    LATEST = "latest"

    def __str__(self) -> str:
        return str(self.value)
