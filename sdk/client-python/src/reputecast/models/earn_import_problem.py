from enum import StrEnum

class EarnImportProblem(StrEnum):
    NO_EMAIL = "no_email"
    VALUE_1 = ""

    def __str__(self) -> str:
        return str(self.value)
