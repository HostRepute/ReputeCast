from enum import StrEnum

class SmsEventKind(StrEnum):
    INBOUND = "inbound"
    OTHER = "other"
    RECEIPT = "receipt"
    TEST = "test"

    def __str__(self) -> str:
        return str(self.value)
