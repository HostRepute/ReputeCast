from enum import StrEnum

class AutomationSMSStepOutside(StrEnum):
    SKIP = "skip"
    VALUE_2 = ""
    WAIT = "wait"

    def __str__(self) -> str:
        return str(self.value)
