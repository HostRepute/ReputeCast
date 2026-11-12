from enum import StrEnum

class ReadinessProgressGroupUnit(StrEnum):
    CONTACTS = "contacts"
    DOMAINS = "domains"
    LINKS = "links"

    def __str__(self) -> str:
        return str(self.value)
