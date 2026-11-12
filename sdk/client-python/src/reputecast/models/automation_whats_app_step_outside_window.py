from enum import StrEnum

class AutomationWhatsAppStepOutsideWindow(StrEnum):
    SKIP = "skip"
    TEMPLATE = "template"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)
