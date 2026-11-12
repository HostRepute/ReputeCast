from enum import StrEnum

class AccessibilityFixInputRole(StrEnum):
    H1 = "h1"
    H2 = "h2"
    H3 = "h3"
    H4 = "h4"
    H5 = "h5"
    H6 = "h6"
    P = "p"

    def __str__(self) -> str:
        return str(self.value)
