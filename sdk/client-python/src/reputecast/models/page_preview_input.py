from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="PagePreviewInput")



@_attrs_define
class PagePreviewInput:
    """ 
        Attributes:
            hide_header (bool | Unset):
            html (str | Unset):
            live (bool | Unset):
            name (str | Unset):
     """

    hide_header: bool | Unset = UNSET
    html: str | Unset = UNSET
    live: bool | Unset = UNSET
    name: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        hide_header = self.hide_header

        html = self.html

        live = self.live

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if hide_header is not UNSET:
            field_dict["hideHeader"] = hide_header
        if html is not UNSET:
            field_dict["html"] = html
        if live is not UNSET:
            field_dict["live"] = live
        if name is not UNSET:
            field_dict["name"] = name

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        hide_header = d.pop("hideHeader", UNSET)

        html = d.pop("html", UNSET)

        live = d.pop("live", UNSET)

        name = d.pop("name", UNSET)

        page_preview_input = cls(
            hide_header=hide_header,
            html=html,
            live=live,
            name=name,
        )

        return page_preview_input

