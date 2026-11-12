from enum import StrEnum

class AIProviderEntryCapabilitiesItem(StrEnum):
    DATA_REGION = "data_region"
    IMAGE_INPUT = "image_input"
    OWN_NETWORK = "own_network"
    STRUCTURED_OUTPUT = "structured_output"
    TEXT = "text"

    def __str__(self) -> str:
        return str(self.value)
