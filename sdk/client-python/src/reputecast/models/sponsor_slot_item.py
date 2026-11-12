from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SponsorSlotItem")



@_attrs_define
class SponsorSlotItem:
    """ 
        Attributes:
            bookings (int):
            creative_days (int):
            currency (str):
            every_days (int):
            first_number (int):
            first_on (datetime.datetime):
            id (str):
            list_id (str):
            list_name (str):
            name (str):
            placement (str):
            price_cents (int):
            selling (bool):
            weekday (str | Unset):
     """

    bookings: int
    creative_days: int
    currency: str
    every_days: int
    first_number: int
    first_on: datetime.datetime
    id: str
    list_id: str
    list_name: str
    name: str
    placement: str
    price_cents: int
    selling: bool
    weekday: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        bookings = self.bookings

        creative_days = self.creative_days

        currency = self.currency

        every_days = self.every_days

        first_number = self.first_number

        first_on = self.first_on.isoformat()

        id = self.id

        list_id = self.list_id

        list_name = self.list_name

        name = self.name

        placement = self.placement

        price_cents = self.price_cents

        selling = self.selling

        weekday = self.weekday


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "bookings": bookings,
            "creativeDays": creative_days,
            "currency": currency,
            "everyDays": every_days,
            "firstNumber": first_number,
            "firstOn": first_on,
            "id": id,
            "listId": list_id,
            "listName": list_name,
            "name": name,
            "placement": placement,
            "priceCents": price_cents,
            "selling": selling,
        })
        if weekday is not UNSET:
            field_dict["weekday"] = weekday

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        bookings = d.pop("bookings")

        creative_days = d.pop("creativeDays")

        currency = d.pop("currency")

        every_days = d.pop("everyDays")

        first_number = d.pop("firstNumber")

        first_on = datetime.datetime.fromisoformat(d.pop("firstOn"))




        id = d.pop("id")

        list_id = d.pop("listId")

        list_name = d.pop("listName")

        name = d.pop("name")

        placement = d.pop("placement")

        price_cents = d.pop("priceCents")

        selling = d.pop("selling")

        weekday = d.pop("weekday", UNSET)

        sponsor_slot_item = cls(
            bookings=bookings,
            creative_days=creative_days,
            currency=currency,
            every_days=every_days,
            first_number=first_number,
            first_on=first_on,
            id=id,
            list_id=list_id,
            list_name=list_name,
            name=name,
            placement=placement,
            price_cents=price_cents,
            selling=selling,
            weekday=weekday,
        )

        return sponsor_slot_item

