from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="DigestSubject")



@_attrs_define
class DigestSubject:
    """ 
        Attributes:
            items (int):
            text (str):
            zero_more (bool):
     """

    items: int
    text: str
    zero_more: bool





    def to_dict(self) -> dict[str, Any]:
        items = self.items

        text = self.text

        zero_more = self.zero_more


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "items": items,
            "text": text,
            "zeroMore": zero_more,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        items = d.pop("items")

        text = d.pop("text")

        zero_more = d.pop("zeroMore")

        digest_subject = cls(
            items=items,
            text=text,
            zero_more=zero_more,
        )

        return digest_subject

