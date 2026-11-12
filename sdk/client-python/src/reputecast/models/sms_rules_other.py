from enum import StrEnum

class SmsRulesOther(StrEnum):
    DONT_SEND = "dont_send"
    ROUTE = "route"

    def __str__(self) -> str:
        return str(self.value)
