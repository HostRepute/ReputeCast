from enum import StrEnum

class SetSidebarSectionSection(StrEnum):
    EARN = "earn"
    GROW = "grow"
    TRANSACTIONAL = "transactional"

    def __str__(self) -> str:
        return str(self.value)
