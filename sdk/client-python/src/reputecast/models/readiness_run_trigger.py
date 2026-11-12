from enum import StrEnum

class ReadinessRunTrigger(StrEnum):
    APPROVAL = "approval"
    COMPOSER = "composer"
    MANUAL = "manual"
    SCHEDULE = "schedule"
    SEND_TIME = "send_time"

    def __str__(self) -> str:
        return str(self.value)
