from enum import StrEnum

class DeliverabilityDMARCExternalRecordType(StrEnum):
    TXT = "TXT"

    def __str__(self) -> str:
        return str(self.value)
