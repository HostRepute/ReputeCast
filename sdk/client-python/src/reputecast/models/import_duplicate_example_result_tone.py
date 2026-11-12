from enum import StrEnum

class ImportDuplicateExampleResultTone(StrEnum):
    BAD = "bad"
    OK = "ok"

    def __str__(self) -> str:
        return str(self.value)
