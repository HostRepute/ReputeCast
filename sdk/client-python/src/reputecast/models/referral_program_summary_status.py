from enum import StrEnum

class ReferralProgramSummaryStatus(StrEnum):
    OFF = "off"
    ON = "on"

    def __str__(self) -> str:
        return str(self.value)
