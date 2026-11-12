from enum import StrEnum

class VerificationStartInputOrigin(StrEnum):
    CONTACTS = "contacts"
    GROWTH = "growth"
    IMPORT = "import"
    LIST_HEALTH = "list_health"
    PROFILE = "profile"
    READINESS = "readiness"
    SEGMENTS = "segments"
    VALUE_7 = ""

    def __str__(self) -> str:
        return str(self.value)
