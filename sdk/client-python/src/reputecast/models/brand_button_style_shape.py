from enum import StrEnum

class BrandButtonStyleShape(StrEnum):
    PILL = "pill"
    ROUNDED = "rounded"
    SQUARE = "square"

    def __str__(self) -> str:
        return str(self.value)
