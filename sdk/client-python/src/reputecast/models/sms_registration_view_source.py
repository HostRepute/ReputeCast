from enum import StrEnum

class SmsRegistrationViewSource(StrEnum):
    API = "api"
    MANUAL = "manual"
    RULE = "rule"
    VALUE_3 = ""

    def __str__(self) -> str:
        return str(self.value)
