from enum import StrEnum

class ImportPreviewOnConflict(StrEnum):
    FILL = "fill"
    KEEP = "keep"
    OVERWRITE = "overwrite"

    def __str__(self) -> str:
        return str(self.value)
