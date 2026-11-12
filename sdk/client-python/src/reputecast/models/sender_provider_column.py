from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SenderProviderColumn")



@_attrs_define
class SenderProviderColumn:
    """ 
        Attributes:
            adapter (str):
            id (str):
            name (str):
            short (str):
     """

    adapter: str
    id: str
    name: str
    short: str





    def to_dict(self) -> dict[str, Any]:
        adapter = self.adapter

        id = self.id

        name = self.name

        short = self.short


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
            "id": id,
            "name": name,
            "short": short,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        adapter = d.pop("adapter")

        id = d.pop("id")

        name = d.pop("name")

        short = d.pop("short")

        sender_provider_column = cls(
            adapter=adapter,
            id=id,
            name=name,
            short=short,
        )

        return sender_provider_column

