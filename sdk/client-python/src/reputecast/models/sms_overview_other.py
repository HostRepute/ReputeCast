from enum import StrEnum

class SmsOverviewOther(StrEnum):
    DONT_SEND = "dont_send"
    ROUTE = "route"

    def __str__(self) -> str:
        return str(self.value)
