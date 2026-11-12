from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="BlockSegment")



@_attrs_define
class BlockSegment:
    """ 
        Attributes:
            column (int):
            heading (int):
            kind (str):
            path (list[int]):
            strong (bool):
            text (str):
            alt (str | Unset):
            html (str | Unset):
            url (str | Unset):
     """

    column: int
    heading: int
    kind: str
    path: list[int]
    strong: bool
    text: str
    alt: str | Unset = UNSET
    html: str | Unset = UNSET
    url: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        column = self.column

        heading = self.heading

        kind = self.kind

        path = self.path



        strong = self.strong

        text = self.text

        alt = self.alt

        html = self.html

        url = self.url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "column": column,
            "heading": heading,
            "kind": kind,
            "path": path,
            "strong": strong,
            "text": text,
        })
        if alt is not UNSET:
            field_dict["alt"] = alt
        if html is not UNSET:
            field_dict["html"] = html
        if url is not UNSET:
            field_dict["url"] = url

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        column = d.pop("column")

        heading = d.pop("heading")

        kind = d.pop("kind")

        path = cast(list[int], d.pop("path"))


        strong = d.pop("strong")

        text = d.pop("text")

        alt = d.pop("alt", UNSET)

        html = d.pop("html", UNSET)

        url = d.pop("url", UNSET)

        block_segment = cls(
            column=column,
            heading=heading,
            kind=kind,
            path=path,
            strong=strong,
            text=text,
            alt=alt,
            html=html,
            url=url,
        )

        return block_segment

