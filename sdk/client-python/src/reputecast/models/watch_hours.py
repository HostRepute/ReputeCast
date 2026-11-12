from enum import IntEnum

class WatchHours(IntEnum):
    VALUE_24 = 24
    VALUE_48 = 48
    VALUE_72 = 72
    VALUE_168 = 168

    def __str__(self) -> str:
        return str(self.value)
