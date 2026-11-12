from enum import StrEnum

class SmsContactChannelsLineSource(StrEnum):
    NUMBER_PLAN = "number_plan"
    PROVIDER = "provider"
    RECEIPT = "receipt"
    VALUE_3 = ""

    def __str__(self) -> str:
        return str(self.value)
