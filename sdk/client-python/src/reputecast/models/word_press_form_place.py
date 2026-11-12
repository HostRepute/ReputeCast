from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="WordPressFormPlace")



@_attrs_define
class WordPressFormPlace:
    """ 
        Attributes:
            title (str):  Example: Join the beta.
            url (str | Unset):
     """

    title: str
    url: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        title = self.title

        url = self.url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "title": title,
        })
        if url is not UNSET:
            field_dict["url"] = url

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        title = d.pop("title")

        url = d.pop("url", UNSET)

        word_press_form_place = cls(
            title=title,
            url=url,
        )

        return word_press_form_place

