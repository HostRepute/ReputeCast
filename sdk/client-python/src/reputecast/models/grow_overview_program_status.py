from enum import StrEnum

class GrowOverviewProgramStatus(StrEnum):
    OFF = "off"
    ON = "on"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)
