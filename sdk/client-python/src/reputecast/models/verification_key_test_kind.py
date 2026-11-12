from enum import StrEnum

class VerificationKeyTestKind(StrEnum):
    KEY_REJECTED = "key_rejected"
    NO_CREDITS = "no_credits"
    UNREACHABLE = "unreachable"
    VALUE_3 = ""

    def __str__(self) -> str:
        return str(self.value)
