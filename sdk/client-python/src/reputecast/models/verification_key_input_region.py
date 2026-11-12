from enum import StrEnum

class VerificationKeyInputRegion(StrEnum):
    EU = "eu"
    US = "us"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)
