from enum import StrEnum

class SignupFormKind(StrEnum):
    EMBEDDED = "embedded"
    HOSTED = "hosted"
    POPUP = "popup"
    PUSH = "push"

    def __str__(self) -> str:
        return str(self.value)
