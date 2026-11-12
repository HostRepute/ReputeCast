from enum import StrEnum

class SmsSendViewPausedKind(StrEnum):
    AUTO = "auto"
    BUDGET = "budget"
    OUTAGE = "outage"
    PERSON = "person"
    VALUE_4 = ""

    def __str__(self) -> str:
        return str(self.value)
