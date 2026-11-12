from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.website_home_section_kind import WebsiteHomeSectionKind
from ..types import UNSET, Unset






T = TypeVar("T", bound="WebsiteHomeSection")



@_attrs_define
class WebsiteHomeSection:
    """ 
        Attributes:
            kind (WebsiteHomeSectionKind):
            on (bool):
            count (int | Unset):
            note (str | Unset):
            tag (str | Unset):
            text (str | Unset):
     """

    kind: WebsiteHomeSectionKind
    on: bool
    count: int | Unset = UNSET
    note: str | Unset = UNSET
    tag: str | Unset = UNSET
    text: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind.value

        on = self.on

        count = self.count

        note = self.note

        tag = self.tag

        text = self.text


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
            "on": on,
        })
        if count is not UNSET:
            field_dict["count"] = count
        if note is not UNSET:
            field_dict["note"] = note
        if tag is not UNSET:
            field_dict["tag"] = tag
        if text is not UNSET:
            field_dict["text"] = text

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = WebsiteHomeSectionKind(d.pop("kind"))




        on = d.pop("on")

        count = d.pop("count", UNSET)

        note = d.pop("note", UNSET)

        tag = d.pop("tag", UNSET)

        text = d.pop("text", UNSET)

        website_home_section = cls(
            kind=kind,
            on=on,
            count=count,
            note=note,
            tag=tag,
            text=text,
        )

        return website_home_section

