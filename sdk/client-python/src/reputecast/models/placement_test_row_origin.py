from enum import StrEnum

class PlacementTestRowOrigin(StrEnum):
    COMPOSER = "composer"
    PLACEMENT = "placement"
    REVIEW = "review"
    TEMPLATE = "template"
    WARMUP = "warmup"

    def __str__(self) -> str:
        return str(self.value)
