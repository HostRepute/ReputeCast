from enum import StrEnum

class PageCheckKey(StrEnum):
    ADDRESS = "address"
    FORM = "form"
    HEADING = "heading"
    IMAGES = "images"
    LINKS = "links"
    MOMENTS = "moments"
    PRODUCT = "product"
    STORE_BLOCKS = "store_blocks"
    STRIPE = "stripe"
    VISIBILITY = "visibility"

    def __str__(self) -> str:
        return str(self.value)
