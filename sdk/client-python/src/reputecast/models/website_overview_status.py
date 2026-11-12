from enum import StrEnum

class WebsiteOverviewStatus(StrEnum):
    OFF = "off"
    ON = "on"
    TURNING_ON = "turning_on"

    def __str__(self) -> str:
        return str(self.value)
