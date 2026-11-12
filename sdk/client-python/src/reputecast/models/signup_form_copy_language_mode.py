from enum import StrEnum

class SignupFormCopyLanguageMode(StrEnum):
    BROWSER = "browser"
    FIXED = "fixed"
    PAGE = "page"

    def __str__(self) -> str:
        return str(self.value)
