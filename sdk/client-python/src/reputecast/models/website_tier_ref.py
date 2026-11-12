from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WebsiteTierRef")



@_attrs_define
class WebsiteTierRef:
    """ 
        Attributes:
            id (str):
            label (str):
            name (str):
     """

    id: str
    label: str
    name: str





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        label = self.label

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "label": label,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        label = d.pop("label")

        name = d.pop("name")

        website_tier_ref = cls(
            id=id,
            label=label,
            name=name,
        )

        return website_tier_ref

