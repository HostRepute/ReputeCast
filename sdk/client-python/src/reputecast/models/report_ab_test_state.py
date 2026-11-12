from enum import StrEnum

class ReportABTestState(StrEnum):
    DECIDED = "decided"
    NO_WINNER = "no_winner"
    TESTING = "testing"

    def __str__(self) -> str:
        return str(self.value)
