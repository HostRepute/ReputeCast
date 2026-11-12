from enum import StrEnum

class SignupFormCreateInputKind(StrEnum):
    EMBEDDED = "embedded"
    HOSTED = "hosted"
    POPUP = "popup"

    def __str__(self) -> str:
        return str(self.value)
