from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ContactActivityProvider")



@_attrs_define
class ContactActivityProvider:
    """ 
        Attributes:
            adapter (str):
            name (str):
     """

    adapter: str
    name: str





    def to_dict(self) -> dict[str, Any]:
        adapter = self.adapter

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        adapter = d.pop("adapter")

        name = d.pop("name")

        contact_activity_provider = cls(
            adapter=adapter,
            name=name,
        )

        return contact_activity_provider

