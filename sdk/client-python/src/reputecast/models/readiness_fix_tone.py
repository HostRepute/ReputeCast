from enum import StrEnum

class ReadinessFixTone(StrEnum):
    DANGER = "danger"
    PRIMARY = "primary"

    def __str__(self) -> str:
        return str(self.value)
