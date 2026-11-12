from enum import StrEnum

class RoleImpactMemberTwoFactor(StrEnum):
    IDP = "idp"
    REQUIRED = "required"

    def __str__(self) -> str:
        return str(self.value)
