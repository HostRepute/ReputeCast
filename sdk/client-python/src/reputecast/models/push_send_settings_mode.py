from enum import StrEnum

class PushSendSettingsMode(StrEnum):
    AT = "at"
    NOW = "now"

    def __str__(self) -> str:
        return str(self.value)
