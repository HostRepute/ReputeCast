from enum import StrEnum

class SignupFormCopyLanguageStatus(StrEnum):
    INCOMPLETE = "incomplete"
    LIVE = "live"
    NOT_SET_UP = "not_set_up"

    def __str__(self) -> str:
        return str(self.value)
