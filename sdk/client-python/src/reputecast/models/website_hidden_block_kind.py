from enum import StrEnum

class WebsiteHiddenBlockKind(StrEnum):
    BLOCK = "block"
    FOOTER = "footer"
    HEADER = "header"
    POLL = "poll"
    REFERRAL = "referral"
    VIEW_ONLINE = "view_online"

    def __str__(self) -> str:
        return str(self.value)
