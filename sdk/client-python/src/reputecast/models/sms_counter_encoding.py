from enum import StrEnum

class SmsCounterEncoding(StrEnum):
    GSM7 = "gsm7"
    UCS2 = "ucs2"

    def __str__(self) -> str:
        return str(self.value)
