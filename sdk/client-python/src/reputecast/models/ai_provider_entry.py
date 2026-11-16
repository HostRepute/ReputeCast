from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.ai_provider_entry_capabilities_item import AIProviderEntryCapabilitiesItem
from ..models.ai_provider_entry_kind import AIProviderEntryKind
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.ai_model_option import AIModelOption
  from ..models.ai_provider_preset import AIProviderPreset





T = TypeVar("T", bound="AIProviderEntry")



@_attrs_define
class AIProviderEntry:
    """ 
        Attributes:
            capabilities (list[AIProviderEntryCapabilitiesItem]):
            kind (AIProviderEntryKind):
            models (list[AIModelOption]):
            name (str):
            needs_base_url (bool):
            needs_key (bool):
            presets (list[AIProviderPreset]):
            default_model (str | Unset):
     """

    capabilities: list[AIProviderEntryCapabilitiesItem]
    kind: AIProviderEntryKind
    models: list[AIModelOption]
    name: str
    needs_base_url: bool
    needs_key: bool
    presets: list[AIProviderPreset]
    default_model: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.ai_model_option import AIModelOption # noqa: PLC0415
        from ..models.ai_provider_preset import AIProviderPreset # noqa: PLC0415
        capabilities = []
        for capabilities_item_data in self.capabilities:
            capabilities_item = capabilities_item_data.value
            capabilities.append(capabilities_item)



        kind = self.kind.value

        models = []
        for models_item_data in self.models:
            models_item = models_item_data.to_dict()
            models.append(models_item)



        name = self.name

        needs_base_url = self.needs_base_url

        needs_key = self.needs_key

        presets = []
        for presets_item_data in self.presets:
            presets_item = presets_item_data.to_dict()
            presets.append(presets_item)



        default_model = self.default_model


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "capabilities": capabilities,
            "kind": kind,
            "models": models,
            "name": name,
            "needsBaseUrl": needs_base_url,
            "needsKey": needs_key,
            "presets": presets,
        })
        if default_model is not UNSET:
            field_dict["defaultModel"] = default_model

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.ai_model_option import AIModelOption # noqa: PLC0415
        from ..models.ai_provider_preset import AIProviderPreset # noqa: PLC0415
        d = dict(src_dict)
        capabilities = []
        _capabilities = d.pop("capabilities")
        for capabilities_item_data in (_capabilities):
            capabilities_item = AIProviderEntryCapabilitiesItem(capabilities_item_data)



            capabilities.append(capabilities_item)


        kind = AIProviderEntryKind(d.pop("kind"))




        models = []
        _models = d.pop("models")
        for models_item_data in (_models):
            models_item = AIModelOption.from_dict(models_item_data)



            models.append(models_item)


        name = d.pop("name")

        needs_base_url = d.pop("needsBaseUrl")

        needs_key = d.pop("needsKey")

        presets = []
        _presets = d.pop("presets")
        for presets_item_data in (_presets):
            presets_item = AIProviderPreset.from_dict(presets_item_data)



            presets.append(presets_item)


        default_model = d.pop("defaultModel", UNSET)

        ai_provider_entry = cls(
            capabilities=capabilities,
            kind=kind,
            models=models,
            name=name,
            needs_base_url=needs_base_url,
            needs_key=needs_key,
            presets=presets,
            default_model=default_model,
        )

        return ai_provider_entry

