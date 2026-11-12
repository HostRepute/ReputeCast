from enum import StrEnum

class VerificationConnectInputRegion(StrEnum):
    EU = "eu"
    US = "us"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)
