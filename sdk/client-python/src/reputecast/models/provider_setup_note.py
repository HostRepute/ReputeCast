from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ProviderSetupNote")



@_attrs_define
class ProviderSetupNote:
    """ 
        Attributes:
            provider (str):
            provider_id (str):
            text (str):
     """

    provider: str
    provider_id: str
    text: str





    def to_dict(self) -> dict[str, Any]:
        provider = self.provider

        provider_id = self.provider_id

        text = self.text


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "provider": provider,
            "providerId": provider_id,
            "text": text,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        provider = d.pop("provider")

        provider_id = d.pop("providerId")

        text = d.pop("text")

        provider_setup_note = cls(
            provider=provider,
            provider_id=provider_id,
            text=text,
        )

        return provider_setup_note

