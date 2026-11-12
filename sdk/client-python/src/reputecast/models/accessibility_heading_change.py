from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="AccessibilityHeadingChange")



@_attrs_define
class AccessibilityHeadingChange:
    """ 
        Attributes:
            from_ (int):
            index (int): The nth heading of the text block, from 0
            path (list[int]):
            text (str):
            to (int):
     """

    from_: int
    index: int
    path: list[int]
    text: str
    to: int





    def to_dict(self) -> dict[str, Any]:
        from_ = self.from_

        index = self.index

        path = self.path



        text = self.text

        to = self.to


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "from": from_,
            "index": index,
            "path": path,
            "text": text,
            "to": to,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        from_ = d.pop("from")

        index = d.pop("index")

        path = cast(list[int], d.pop("path"))


        text = d.pop("text")

        to = d.pop("to")

        accessibility_heading_change = cls(
            from_=from_,
            index=index,
            path=path,
            text=text,
            to=to,
        )

        return accessibility_heading_change

