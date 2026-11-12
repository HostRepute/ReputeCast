from enum import StrEnum

class GrowPreviewContactsStatus(StrEnum):
    OFF = "off"
    ON = "on"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)
