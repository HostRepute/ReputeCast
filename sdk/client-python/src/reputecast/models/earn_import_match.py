from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="EarnImportMatch")



@_attrs_define
class EarnImportMatch:
    """ 
        Attributes:
            email (str):
            row (int):
            amount_cents (int | Unset):
            currency (str | Unset):
            customer (str | Unset):
            ended_at (datetime.datetime | Unset):
            name (str | Unset):
            paid_through (datetime.datetime | Unset):
            period (str | Unset):
            tier (str | Unset):
     """

    email: str
    row: int
    amount_cents: int | Unset = UNSET
    currency: str | Unset = UNSET
    customer: str | Unset = UNSET
    ended_at: datetime.datetime | Unset = UNSET
    name: str | Unset = UNSET
    paid_through: datetime.datetime | Unset = UNSET
    period: str | Unset = UNSET
    tier: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        email = self.email

        row = self.row

        amount_cents = self.amount_cents

        currency = self.currency

        customer = self.customer

        ended_at: str | Unset = UNSET
        if not isinstance(self.ended_at, Unset):
            ended_at = self.ended_at.isoformat()

        name = self.name

        paid_through: str | Unset = UNSET
        if not isinstance(self.paid_through, Unset):
            paid_through = self.paid_through.isoformat()

        period = self.period

        tier = self.tier


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "email": email,
            "row": row,
        })
        if amount_cents is not UNSET:
            field_dict["amountCents"] = amount_cents
        if currency is not UNSET:
            field_dict["currency"] = currency
        if customer is not UNSET:
            field_dict["customer"] = customer
        if ended_at is not UNSET:
            field_dict["endedAt"] = ended_at
        if name is not UNSET:
            field_dict["name"] = name
        if paid_through is not UNSET:
            field_dict["paidThrough"] = paid_through
        if period is not UNSET:
            field_dict["period"] = period
        if tier is not UNSET:
            field_dict["tier"] = tier

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        email = d.pop("email")

        row = d.pop("row")

        amount_cents = d.pop("amountCents", UNSET)

        currency = d.pop("currency", UNSET)

        customer = d.pop("customer", UNSET)

        _ended_at = d.pop("endedAt", UNSET)
        ended_at: datetime.datetime | Unset
        if isinstance(_ended_at,  Unset):
            ended_at = UNSET
        else:
            ended_at = datetime.datetime.fromisoformat(_ended_at)




        name = d.pop("name", UNSET)

        _paid_through = d.pop("paidThrough", UNSET)
        paid_through: datetime.datetime | Unset
        if isinstance(_paid_through,  Unset):
            paid_through = UNSET
        else:
            paid_through = datetime.datetime.fromisoformat(_paid_through)




        period = d.pop("period", UNSET)

        tier = d.pop("tier", UNSET)

        earn_import_match = cls(
            email=email,
            row=row,
            amount_cents=amount_cents,
            currency=currency,
            customer=customer,
            ended_at=ended_at,
            name=name,
            paid_through=paid_through,
            period=period,
            tier=tier,
        )

        return earn_import_match

