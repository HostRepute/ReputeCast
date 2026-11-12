from enum import StrEnum

class SettingsProviderCardStatusTone(StrEnum):
    BAD = "bad"
    INFO = "info"
    NEUTRAL = "neutral"
    OK = "ok"
    WARN = "warn"

    def __str__(self) -> str:
        return str(self.value)
