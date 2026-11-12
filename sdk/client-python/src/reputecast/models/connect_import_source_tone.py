from enum import StrEnum

class ConnectImportSourceTone(StrEnum):
    NEUTRAL = "neutral"
    OK = "ok"

    def __str__(self) -> str:
        return str(self.value)
