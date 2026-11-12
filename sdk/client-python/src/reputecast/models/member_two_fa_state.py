from enum import StrEnum

class MemberTwoFAState(StrEnum):
    IDP = "idp"
    OFF = "off"
    ON = "on"
    REQUIRED = "required"

    def __str__(self) -> str:
        return str(self.value)
