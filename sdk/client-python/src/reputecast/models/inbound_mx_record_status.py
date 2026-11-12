from enum import StrEnum

class InboundMXRecordStatus(StrEnum):
    FOUND = "found"
    MISSING = "missing"
    UNKNOWN = "unknown"
    WRONG_VALUE = "wrong_value"

    def __str__(self) -> str:
        return str(self.value)
