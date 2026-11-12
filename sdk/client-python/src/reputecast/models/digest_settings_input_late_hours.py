from enum import IntEnum

class DigestSettingsInputLateHours(IntEnum):
    VALUE_6 = 6
    VALUE_24 = 24
    VALUE_48 = 48

    def __str__(self) -> str:
        return str(self.value)
