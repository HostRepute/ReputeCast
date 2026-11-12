from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="SmsCampaignUpdate")



@_attrs_define
class SmsCampaignUpdate:
    """ 
        Attributes:
            version (int):
            body (str | Unset):
            lists (list[str] | Unset):
            name (str | Unset):
            segments (list[str] | Unset):
     """

    version: int
    body: str | Unset = UNSET
    lists: list[str] | Unset = UNSET
    name: str | Unset = UNSET
    segments: list[str] | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        version = self.version

        body = self.body

        lists: list[str] | Unset = UNSET
        if not isinstance(self.lists, Unset):
            lists = self.lists



        name = self.name

        segments: list[str] | Unset = UNSET
        if not isinstance(self.segments, Unset):
            segments = self.segments




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "version": version,
        })
        if body is not UNSET:
            field_dict["body"] = body
        if lists is not UNSET:
            field_dict["lists"] = lists
        if name is not UNSET:
            field_dict["name"] = name
        if segments is not UNSET:
            field_dict["segments"] = segments

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        version = d.pop("version")

        body = d.pop("body", UNSET)

        lists = cast(list[str], d.pop("lists", UNSET))


        name = d.pop("name", UNSET)

        segments = cast(list[str], d.pop("segments", UNSET))


        sms_campaign_update = cls(
            version=version,
            body=body,
            lists=lists,
            name=name,
            segments=segments,
        )

        return sms_campaign_update

