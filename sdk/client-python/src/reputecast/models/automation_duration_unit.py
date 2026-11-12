from enum import StrEnum

class AutomationDurationUnit(StrEnum):
    DAYS = "days"
    HOURS = "hours"
    MINUTES = "minutes"
    WEEKS = "weeks"

    def __str__(self) -> str:
        return str(self.value)
