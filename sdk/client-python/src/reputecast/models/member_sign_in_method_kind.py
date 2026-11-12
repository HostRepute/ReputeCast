from enum import StrEnum

class MemberSignInMethodKind(StrEnum):
    GITHUB = "github"
    GOOGLE = "google"
    MICROSOFT = "microsoft"
    PASSKEY = "passkey"
    PASSWORD = "password"
    SSO = "sso"

    def __str__(self) -> str:
        return str(self.value)
