from enum import StrEnum

class ConnectCatalogEntryBulkMark(StrEnum):
    LIMITED = "limited"
    NO = "no"
    YES = "yes"

    def __str__(self) -> str:
        return str(self.value)
