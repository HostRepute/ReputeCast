from enum import StrEnum

class IntegrationCallViewWho(StrEnum):
    CONTACT = "contact"
    EMAIL = "email"
    ERASED = "erased"
    NONE = "none"

    def __str__(self) -> str:
        return str(self.value)
