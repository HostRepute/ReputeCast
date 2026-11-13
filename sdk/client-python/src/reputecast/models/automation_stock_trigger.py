from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="AutomationStockTrigger")



@_attrs_define
class AutomationStockTrigger:
    """ 
        Attributes:
            store_id (str): The connected store (uuid) whose products it watches
            days (int | Unset):  Default: 14.
            include_viewers (bool | Unset):
            min_drop_percent (int | Unset):  Default: 10.
     """

    store_id: str
    days: int | Unset = 14
    include_viewers: bool | Unset = UNSET
    min_drop_percent: int | Unset = 10





    def to_dict(self) -> dict[str, Any]:
        store_id = self.store_id

        days = self.days

        include_viewers = self.include_viewers

        min_drop_percent = self.min_drop_percent


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "storeId": store_id,
        })
        if days is not UNSET:
            field_dict["days"] = days
        if include_viewers is not UNSET:
            field_dict["includeViewers"] = include_viewers
        if min_drop_percent is not UNSET:
            field_dict["minDropPercent"] = min_drop_percent

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        store_id = d.pop("storeId")

        days = d.pop("days", UNSET)

        include_viewers = d.pop("includeViewers", UNSET)

        min_drop_percent = d.pop("minDropPercent", UNSET)

        automation_stock_trigger = cls(
            store_id=store_id,
            days=days,
            include_viewers=include_viewers,
            min_drop_percent=min_drop_percent,
        )

        return automation_stock_trigger

