from enum import StrEnum

class WebsiteAccessComments(StrEnum):
    MEMBERS = "members"
    OFF = "off"
    PAID = "paid"

    def __str__(self) -> str:
        return str(self.value)
