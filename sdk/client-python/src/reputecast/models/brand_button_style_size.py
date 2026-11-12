from enum import IntEnum

class BrandButtonStyleSize(IntEnum):
    VALUE_14 = 14
    VALUE_15 = 15

    def __str__(self) -> str:
        return str(self.value)
