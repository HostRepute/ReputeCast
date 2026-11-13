from enum import StrEnum

class SignupFormFieldType(StrEnum):
    CHOICE = "choice"

    def __str__(self) -> str:
        return str(self.value)
