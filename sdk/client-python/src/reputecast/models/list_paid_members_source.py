from enum import StrEnum

class ListPaidMembersSource(StrEnum):
    AUTOMATION = "automation"
    CAMPAIGN = "campaign"
    DIRECT = "direct"
    GIFT = "gift"
    GRANT = "grant"
    IMPORT = "import"
    OFFER = "offer"
    PAGE = "page"
    VALUE_9 = ""
    WEBSITE = "website"

    def __str__(self) -> str:
        return str(self.value)
