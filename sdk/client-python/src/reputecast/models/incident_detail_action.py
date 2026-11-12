from enum import StrEnum

class IncidentDetailAction(StrEnum):
    ALERTED = "alerted"
    SLOWED = "slowed"
    VALUE_3 = ""
    WARNED = "warned"

    def __str__(self) -> str:
        return str(self.value)
