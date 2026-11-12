from enum import StrEnum

class SignupFormAfterSubmit(StrEnum):
    MESSAGE = "message"
    REDIRECT = "redirect"

    def __str__(self) -> str:
        return str(self.value)
