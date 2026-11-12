from enum import StrEnum

class AudienceJobKind(StrEnum):
    EXPORT = "export"
    RESEND_OPTIN = "resend_optin"

    def __str__(self) -> str:
        return str(self.value)
