from enum import StrEnum

class PlacementRunInputOrigin(StrEnum):
    COMPOSER = "composer"
    PLACEMENT = "placement"
    REVIEW = "review"
    TEMPLATE = "template"
    VALUE_5 = ""
    WARMUP = "warmup"

    def __str__(self) -> str:
        return str(self.value)
