from enum import StrEnum

class SignupFormABResultOutcome(StrEnum):
    NOT_SIGNIFICANT = "not_significant"
    TOO_SMALL = "too_small"
    WINNER = "winner"

    def __str__(self) -> str:
        return str(self.value)
