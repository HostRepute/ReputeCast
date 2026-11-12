from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="SmsCampaignCreate")



@_attrs_define
class SmsCampaignCreate:
    """ 
        Attributes:
            name (str):
            body (str | Unset):
            lists (list[str] | Unset):
     """

    name: str
    body: str | Unset = UNSET
    lists: list[str] | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        body = self.body

        lists: list[str] | Unset = UNSET
        if not isinstance(self.lists, Unset):
            lists = self.lists




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
        })
        if body is not UNSET:
            field_dict["body"] = body
        if lists is not UNSET:
            field_dict["lists"] = lists

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        body = d.pop("body", UNSET)

        lists = cast(list[str], d.pop("lists", UNSET))


        sms_campaign_create = cls(
            name=name,
            body=body,
            lists=lists,
        )

        return sms_campaign_create

