from enum import StrEnum

class GetCampaignSponsorBlockLayout(StrEnum):
    IMAGE_TOP = "image-top"
    LOGO_LEFT = "logo-left"
    TEXT = "text"
    VALUE_3 = ""

    def __str__(self) -> str:
        return str(self.value)
