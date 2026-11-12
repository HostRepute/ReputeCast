from enum import StrEnum

class PlacementSeedViewProvider(StrEnum):
    AOL = "aol"
    GMAIL = "gmail"
    GMX = "gmx"
    GOOGLE_WORKSPACE = "google_workspace"
    ICLOUD = "icloud"
    MICROSOFT_365 = "microsoft_365"
    OUTLOOK = "outlook"
    WEBDE = "webde"
    YAHOO = "yahoo"

    def __str__(self) -> str:
        return str(self.value)
