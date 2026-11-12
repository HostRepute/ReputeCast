from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PublicPagePreview")



@_attrs_define
class PublicPagePreview:
    """ 
        Attributes:
            contact (str):
            html (str):
            url (str):
     """

    contact: str
    html: str
    url: str





    def to_dict(self) -> dict[str, Any]:
        contact = self.contact

        html = self.html

        url = self.url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "contact": contact,
            "html": html,
            "url": url,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        contact = d.pop("contact")

        html = d.pop("html")

        url = d.pop("url")

        public_page_preview = cls(
            contact=contact,
            html=html,
            url=url,
        )

        return public_page_preview

