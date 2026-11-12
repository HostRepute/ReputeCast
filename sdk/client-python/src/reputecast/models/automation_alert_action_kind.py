from enum import StrEnum

class AutomationAlertActionKind(StrEnum):
    OPEN_STEP = "open_step"
    PROVIDERS = "providers"
    ROUTING = "routing"
    STEP_LOG = "step_log"

    def __str__(self) -> str:
        return str(self.value)
