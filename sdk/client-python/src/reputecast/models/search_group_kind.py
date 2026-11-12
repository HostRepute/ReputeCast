from enum import StrEnum

class SearchGroupKind(StrEnum):
    AUTOMATIONS = "automations"
    CAMPAIGNS = "campaigns"
    CONTACTS = "contacts"
    FORMS = "forms"
    LISTS = "lists"
    PAGES = "pages"
    SEGMENTS = "segments"

    def __str__(self) -> str:
        return str(self.value)
