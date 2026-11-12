from enum import StrEnum

class AccessibilityElementRole(StrEnum):
    BUTTON = "button"
    HEADING = "heading"
    IMAGE = "image"
    LINK = "link"
    LINKED_IMAGE = "linked_image"
    TEXT = "text"

    def __str__(self) -> str:
        return str(self.value)
