from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.page_publish_input_when import PagePublishInputWhen
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="PagePublishInput")



@_attrs_define
class PagePublishInput:
    """ 
        Attributes:
            when (PagePublishInputWhen):
            at (datetime.datetime | Unset):
            tz (str | Unset):
     """

    when: PagePublishInputWhen
    at: datetime.datetime | Unset = UNSET
    tz: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        when = self.when.value

        at: str | Unset = UNSET
        if not isinstance(self.at, Unset):
            at = self.at.isoformat()

        tz = self.tz


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "when": when,
        })
        if at is not UNSET:
            field_dict["at"] = at
        if tz is not UNSET:
            field_dict["tz"] = tz

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        when = PagePublishInputWhen(d.pop("when"))




        _at = d.pop("at", UNSET)
        at: datetime.datetime | Unset
        if isinstance(_at,  Unset):
            at = UNSET
        else:
            at = datetime.datetime.fromisoformat(_at)




        tz = d.pop("tz", UNSET)

        page_publish_input = cls(
            when=when,
            at=at,
            tz=tz,
        )

        return page_publish_input

