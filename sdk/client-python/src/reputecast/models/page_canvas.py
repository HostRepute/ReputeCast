from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="PageCanvas")



@_attrs_define
class PageCanvas:
    """ 
        Attributes:
            dir_ (str):
            focus_header (str):
            footer (str):
            header (str):
            lang (str):
            styles (list[str]):
     """

    dir_: str
    focus_header: str
    footer: str
    header: str
    lang: str
    styles: list[str]





    def to_dict(self) -> dict[str, Any]:
        dir_ = self.dir_

        focus_header = self.focus_header

        footer = self.footer

        header = self.header

        lang = self.lang

        styles = self.styles




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "dir": dir_,
            "focusHeader": focus_header,
            "footer": footer,
            "header": header,
            "lang": lang,
            "styles": styles,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        dir_ = d.pop("dir")

        focus_header = d.pop("focusHeader")

        footer = d.pop("footer")

        header = d.pop("header")

        lang = d.pop("lang")

        styles = cast(list[str], d.pop("styles"))


        page_canvas = cls(
            dir_=dir_,
            focus_header=focus_header,
            footer=footer,
            header=header,
            lang=lang,
            styles=styles,
        )

        return page_canvas

