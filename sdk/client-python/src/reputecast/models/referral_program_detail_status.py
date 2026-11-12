from enum import StrEnum

class ReferralProgramDetailStatus(StrEnum):
    OFF = "off"
    ON = "on"

    def __str__(self) -> str:
        return str(self.value)
