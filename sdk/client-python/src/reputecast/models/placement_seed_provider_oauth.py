from enum import StrEnum

class PlacementSeedProviderOauth(StrEnum):
    GOOGLE = "google"
    MICROSOFT = "microsoft"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)
