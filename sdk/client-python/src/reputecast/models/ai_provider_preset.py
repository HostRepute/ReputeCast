from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="AIProviderPreset")



@_attrs_define
class AIProviderPreset:
    """ 
        Attributes:
            base_url (str):
            id (str):
            name (str):
     """

    base_url: str
    id: str
    name: str





    def to_dict(self) -> dict[str, Any]:
        base_url = self.base_url

        id = self.id

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "baseUrl": base_url,
            "id": id,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        base_url = d.pop("baseUrl")

        id = d.pop("id")

        name = d.pop("name")

        ai_provider_preset = cls(
            base_url=base_url,
            id=id,
            name=name,
        )

        return ai_provider_preset

