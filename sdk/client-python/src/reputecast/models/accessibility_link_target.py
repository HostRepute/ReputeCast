from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="AccessibilityLinkTarget")



@_attrs_define
class AccessibilityLinkTarget:
    """ 
        Attributes:
            href (str):
            index (int): The nth link of the text block, from 0
            path (list[int]):
     """

    href: str
    index: int
    path: list[int]





    def to_dict(self) -> dict[str, Any]:
        href = self.href

        index = self.index

        path = self.path




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "href": href,
            "index": index,
            "path": path,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        href = d.pop("href")

        index = d.pop("index")

        path = cast(list[int], d.pop("path"))


        accessibility_link_target = cls(
            href=href,
            index=index,
            path=path,
        )

        return accessibility_link_target

