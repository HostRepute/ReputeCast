from enum import StrEnum

class WebsiteMemberEventKind(StrEnum):
    EMAIL_CHANGED = "email_changed"
    JOINED = "joined"
    LINK_REFUSED = "link_refused"
    SIGNED_IN = "signed_in"
    SIGNED_OUT_EVERYWHERE = "signed_out_everywhere"

    def __str__(self) -> str:
        return str(self.value)
