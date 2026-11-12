from enum import StrEnum

class WarmupSettingsGuardAction(StrEnum):
    HOLD = "hold"
    PAUSE = "pause"
    STEP_BACK = "step_back"

    def __str__(self) -> str:
        return str(self.value)
