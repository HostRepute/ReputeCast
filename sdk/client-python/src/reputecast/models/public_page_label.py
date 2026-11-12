from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="PublicPageLabel")



@_attrs_define
class PublicPageLabel:
    """ 
        Attributes:
            arguments (list[str]):
            key (str):  Example: form.button.
            source (str): The English text
            text (str):
            note (str | Unset):
     """

    arguments: list[str]
    key: str
    source: str
    text: str
    note: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        arguments = self.arguments



        key = self.key

        source = self.source

        text = self.text

        note = self.note


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "arguments": arguments,
            "key": key,
            "source": source,
            "text": text,
        })
        if note is not UNSET:
            field_dict["note"] = note

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        arguments = cast(list[str], d.pop("arguments"))


        key = d.pop("key")

        source = d.pop("source")

        text = d.pop("text")

        note = d.pop("note", UNSET)

        public_page_label = cls(
            arguments=arguments,
            key=key,
            source=source,
            text=text,
            note=note,
        )

        return public_page_label

