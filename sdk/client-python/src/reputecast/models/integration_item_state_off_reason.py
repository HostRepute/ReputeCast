from enum import StrEnum

class IntegrationItemStateOffReason(StrEnum):
    OWNER_LACKS = "owner_lacks"
    TURNED_OFF = "turned_off"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)
