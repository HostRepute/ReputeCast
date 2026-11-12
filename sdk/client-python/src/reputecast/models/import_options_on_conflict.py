from enum import StrEnum

class ImportOptionsOnConflict(StrEnum):
    FILL = "fill"
    KEEP = "keep"
    OVERWRITE = "overwrite"

    def __str__(self) -> str:
        return str(self.value)
