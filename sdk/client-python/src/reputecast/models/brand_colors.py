from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="BrandColors")



@_attrs_define
class BrandColors:
    """ 
        Attributes:
            background (str): Behind the 600 px email column
            button (str): Button fill, with white text
            link (str): Links inside text, underlined
            text (str): Paragraphs and headings
     """

    background: str
    button: str
    link: str
    text: str





    def to_dict(self) -> dict[str, Any]:
        background = self.background

        button = self.button

        link = self.link

        text = self.text


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "background": background,
            "button": button,
            "link": link,
            "text": text,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        background = d.pop("background")

        button = d.pop("button")

        link = d.pop("link")

        text = d.pop("text")

        brand_colors = cls(
            background=background,
            button=button,
            link=link,
            text=text,
        )

        return brand_colors

