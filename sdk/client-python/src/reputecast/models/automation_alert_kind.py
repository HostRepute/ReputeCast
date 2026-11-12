from enum import StrEnum

class AutomationAlertKind(StrEnum):
    SENDING_HELD = "sending_held"
    STEP_FAILING = "step_failing"

    def __str__(self) -> str:
        return str(self.value)
