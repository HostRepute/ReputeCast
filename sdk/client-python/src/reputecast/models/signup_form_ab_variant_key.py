from enum import StrEnum

class SignupFormABVariantKey(StrEnum):
    A = "A"
    B = "B"
    C = "C"

    def __str__(self) -> str:
        return str(self.value)
