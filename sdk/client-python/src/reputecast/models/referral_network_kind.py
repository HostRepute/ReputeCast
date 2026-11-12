from enum import StrEnum

class ReferralNetworkKind(StrEnum):
    HOSTING = "hosting"
    MOBILE = "mobile"
    OFFICE = "office"
    RESIDENTIAL = "residential"
    VALUE_4 = ""

    def __str__(self) -> str:
        return str(self.value)
