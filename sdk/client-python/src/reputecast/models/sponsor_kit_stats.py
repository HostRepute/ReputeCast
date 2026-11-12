from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="SponsorKitStats")



@_attrs_define
class SponsorKitStats:
    """ 
        Attributes:
            list_id (str):
            list_name (str):
            subscribers (int):
            click_rate (float | Unset):
            open_rate (float | Unset):
     """

    list_id: str
    list_name: str
    subscribers: int
    click_rate: float | Unset = UNSET
    open_rate: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        list_id = self.list_id

        list_name = self.list_name

        subscribers = self.subscribers

        click_rate = self.click_rate

        open_rate = self.open_rate


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "listId": list_id,
            "listName": list_name,
            "subscribers": subscribers,
        })
        if click_rate is not UNSET:
            field_dict["clickRate"] = click_rate
        if open_rate is not UNSET:
            field_dict["openRate"] = open_rate

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        list_id = d.pop("listId")

        list_name = d.pop("listName")

        subscribers = d.pop("subscribers")

        click_rate = d.pop("clickRate", UNSET)

        open_rate = d.pop("openRate", UNSET)

        sponsor_kit_stats = cls(
            list_id=list_id,
            list_name=list_name,
            subscribers=subscribers,
            click_rate=click_rate,
            open_rate=open_rate,
        )

        return sponsor_kit_stats

