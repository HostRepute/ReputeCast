from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PlacementOther")



@_attrs_define
class PlacementOther:
    """ 
        Attributes:
            copies (int):
            inbox (int):
            name (str):
            provider_id (str):
            spam (int):
     """

    copies: int
    inbox: int
    name: str
    provider_id: str
    spam: int





    def to_dict(self) -> dict[str, Any]:
        copies = self.copies

        inbox = self.inbox

        name = self.name

        provider_id = self.provider_id

        spam = self.spam


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "copies": copies,
            "inbox": inbox,
            "name": name,
            "providerId": provider_id,
            "spam": spam,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        copies = d.pop("copies")

        inbox = d.pop("inbox")

        name = d.pop("name")

        provider_id = d.pop("providerId")

        spam = d.pop("spam")

        placement_other = cls(
            copies=copies,
            inbox=inbox,
            name=name,
            provider_id=provider_id,
            spam=spam,
        )

        return placement_other

