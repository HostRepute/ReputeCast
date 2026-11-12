from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="PageRendered")



@_attrs_define
class PageRendered:
    """ 
        Attributes:
            blocks (list[str]):
            html (str | Unset):
     """

    blocks: list[str]
    html: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        blocks = self.blocks



        html = self.html


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "blocks": blocks,
        })
        if html is not UNSET:
            field_dict["html"] = html

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        blocks = cast(list[str], d.pop("blocks"))


        html = d.pop("html", UNSET)

        page_rendered = cls(
            blocks=blocks,
            html=html,
        )

        return page_rendered

