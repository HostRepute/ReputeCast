from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="GiftDeliveryInput")



@_attrs_define
class GiftDeliveryInput:
    """ 
        Attributes:
            date (datetime.date): The day, in the time zone
            time (str): The time of day, 24-hour
            notify_buyer (bool | Unset):
            timezone (str | Unset): An IANA time zone; the buyer's is the default
     """

    date: datetime.date
    time: str
    notify_buyer: bool | Unset = UNSET
    timezone: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        date = self.date.isoformat()

        time = self.time

        notify_buyer = self.notify_buyer

        timezone = self.timezone


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "date": date,
            "time": time,
        })
        if notify_buyer is not UNSET:
            field_dict["notifyBuyer"] = notify_buyer
        if timezone is not UNSET:
            field_dict["timezone"] = timezone

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        date = datetime.date.fromisoformat(d.pop("date"))




        time = d.pop("time")

        notify_buyer = d.pop("notifyBuyer", UNSET)

        timezone = d.pop("timezone", UNSET)

        gift_delivery_input = cls(
            date=date,
            time=time,
            notify_buyer=notify_buyer,
            timezone=timezone,
        )

        return gift_delivery_input

