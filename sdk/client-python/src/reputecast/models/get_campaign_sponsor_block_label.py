from enum import StrEnum

class GetCampaignSponsorBlockLabel(StrEnum):
    NAME = "name"
    PLAIN = "plain"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)
