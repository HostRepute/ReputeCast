from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
from uuid import UUID
import datetime






T = TypeVar("T", bound="SlotInput")



@_attrs_define
class SlotInput:
    """ 
        Attributes:
            first_number (int):
            first_on (datetime.date): The date of issue FirstNumber, in the workspace's time zone.
            list_id (UUID):
            name (str):
            price_cents (int): The default price of a booking, in minor units.
            creative_days (int | Unset): The creative is due this many days before the issue (4 when absent).
            currency (str | Unset): ISO 4217; the workspace's currency when absent.
            description (str | Unset): What the sponsor page says about the slot.
            every_days (int | Unset): 7 when absent (weekly).
            lead_days (int | Unset): Sponsors book at least this many days ahead (7 for a new slot when absent).
            pay_days (int | Unset): Sponsors pay within this many days of the approval (3 for a new slot when absent).
            placement (str | Unset): Where it sits in the issue ("After the first story").
            self_serve (bool | Unset): Sponsors can book it from the sponsor page (off for a new slot when absent).
            selling (bool | Unset):
     """

    first_number: int
    first_on: datetime.date
    list_id: UUID
    name: str
    price_cents: int
    creative_days: int | Unset = UNSET
    currency: str | Unset = UNSET
    description: str | Unset = UNSET
    every_days: int | Unset = UNSET
    lead_days: int | Unset = UNSET
    pay_days: int | Unset = UNSET
    placement: str | Unset = UNSET
    self_serve: bool | Unset = UNSET
    selling: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        first_number = self.first_number

        first_on = self.first_on.isoformat()

        list_id = str(self.list_id)

        name = self.name

        price_cents = self.price_cents

        creative_days = self.creative_days

        currency = self.currency

        description = self.description

        every_days = self.every_days

        lead_days = self.lead_days

        pay_days = self.pay_days

        placement = self.placement

        self_serve = self.self_serve

        selling = self.selling


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "firstNumber": first_number,
            "firstOn": first_on,
            "listId": list_id,
            "name": name,
            "priceCents": price_cents,
        })
        if creative_days is not UNSET:
            field_dict["creativeDays"] = creative_days
        if currency is not UNSET:
            field_dict["currency"] = currency
        if description is not UNSET:
            field_dict["description"] = description
        if every_days is not UNSET:
            field_dict["everyDays"] = every_days
        if lead_days is not UNSET:
            field_dict["leadDays"] = lead_days
        if pay_days is not UNSET:
            field_dict["payDays"] = pay_days
        if placement is not UNSET:
            field_dict["placement"] = placement
        if self_serve is not UNSET:
            field_dict["selfServe"] = self_serve
        if selling is not UNSET:
            field_dict["selling"] = selling

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        first_number = d.pop("firstNumber")

        first_on = datetime.date.fromisoformat(d.pop("firstOn"))




        list_id = UUID(d.pop("listId"))




        name = d.pop("name")

        price_cents = d.pop("priceCents")

        creative_days = d.pop("creativeDays", UNSET)

        currency = d.pop("currency", UNSET)

        description = d.pop("description", UNSET)

        every_days = d.pop("everyDays", UNSET)

        lead_days = d.pop("leadDays", UNSET)

        pay_days = d.pop("payDays", UNSET)

        placement = d.pop("placement", UNSET)

        self_serve = d.pop("selfServe", UNSET)

        selling = d.pop("selling", UNSET)

        slot_input = cls(
            first_number=first_number,
            first_on=first_on,
            list_id=list_id,
            name=name,
            price_cents=price_cents,
            creative_days=creative_days,
            currency=currency,
            description=description,
            every_days=every_days,
            lead_days=lead_days,
            pay_days=pay_days,
            placement=placement,
            self_serve=self_serve,
            selling=selling,
        )

        return slot_input

