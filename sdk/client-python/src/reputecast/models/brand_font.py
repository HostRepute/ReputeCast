from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="BrandFont")



@_attrs_define
class BrandFont:
    """ 
        Attributes:
            fallback (str): One of the email-safe stacks: Georgia, 'Times New Roman', serif · 'Times New Roman', serif ·
                Arial, Helvetica, sans-serif · Helvetica, Arial, sans-serif · Verdana, Geneva, sans-serif · 'Trebuchet MS',
                Arial, sans-serif · Tahoma, Geneva, sans-serif
            name (str): The web font's family (Inter); empty uses the fallback alone
            url (str | Unset): The web font's stylesheet (https)
     """

    fallback: str
    name: str
    url: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        fallback = self.fallback

        name = self.name

        url = self.url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "fallback": fallback,
            "name": name,
        })
        if url is not UNSET:
            field_dict["url"] = url

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        fallback = d.pop("fallback")

        name = d.pop("name")

        url = d.pop("url", UNSET)

        brand_font = cls(
            fallback=fallback,
            name=name,
            url=url,
        )

        return brand_font

