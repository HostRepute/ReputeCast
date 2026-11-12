from enum import StrEnum

class ConnectCatalogEntryTermsTone(StrEnum):
    NEUTRAL = "neutral"
    OK = "ok"
    WARN = "warn"

    def __str__(self) -> str:
        return str(self.value)
