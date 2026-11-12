from enum import StrEnum

class AutomationStepContactWait(StrEnum):
    BRANCH = "branch"
    CAP = "cap"
    DELAY = "delay"
    VALUE_5 = ""
    WEBHOOK = "webhook"
    WINDOW = "window"

    def __str__(self) -> str:
        return str(self.value)
