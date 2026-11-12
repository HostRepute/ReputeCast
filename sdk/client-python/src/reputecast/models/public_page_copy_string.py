from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.public_page_copy_string_page import PublicPageCopyStringPage
from ..types import UNSET, Unset






T = TypeVar("T", bound="PublicPageCopyString")



@_attrs_define
class PublicPageCopyString:
    """ 
        Attributes:
            key (str):  Example: subscribe.heading.
            optional (bool):
            page (PublicPageCopyStringPage):
            source (str):
            text (str):
            list_id (str | Unset):
     """

    key: str
    optional: bool
    page: PublicPageCopyStringPage
    source: str
    text: str
    list_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        key = self.key

        optional = self.optional

        page = self.page.value

        source = self.source

        text = self.text

        list_id = self.list_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "optional": optional,
            "page": page,
            "source": source,
            "text": text,
        })
        if list_id is not UNSET:
            field_dict["listId"] = list_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = d.pop("key")

        optional = d.pop("optional")

        page = PublicPageCopyStringPage(d.pop("page"))




        source = d.pop("source")

        text = d.pop("text")

        list_id = d.pop("listId", UNSET)

        public_page_copy_string = cls(
            key=key,
            optional=optional,
            page=page,
            source=source,
            text=text,
            list_id=list_id,
        )

        return public_page_copy_string

