from enum import StrEnum

class PushCheckKey(StrEnum):
    APPROVAL = "approval"
    AUDIENCE = "audience"
    FREQUENCY_CAP = "frequency_cap"
    IMAGE = "image"
    KEYS = "keys"
    LENGTHS = "lengths"
    LINK = "link"
    QUIET_HOURS = "quiet_hours"
    WORKER = "worker"

    def __str__(self) -> str:
        return str(self.value)
