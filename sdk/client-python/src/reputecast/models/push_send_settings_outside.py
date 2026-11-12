from enum import StrEnum

class PushSendSettingsOutside(StrEnum):
    HOLD = "hold"
    SKIP = "skip"

    def __str__(self) -> str:
        return str(self.value)
