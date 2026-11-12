from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="EarnImportCandidate")



@_attrs_define
class EarnImportCandidate:
    """ 
        Attributes:
            created (datetime.datetime):
            customer (str):
            amount_cents (int | Unset):
            currency (str | Unset):
            period (str | Unset):
            since (datetime.datetime | Unset):
            status (str | Unset):
            tier (str | Unset):
     """

    created: datetime.datetime
    customer: str
    amount_cents: int | Unset = UNSET
    currency: str | Unset = UNSET
    period: str | Unset = UNSET
    since: datetime.datetime | Unset = UNSET
    status: str | Unset = UNSET
    tier: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        created = self.created.isoformat()

        customer = self.customer

        amount_cents = self.amount_cents

        currency = self.currency

        period = self.period

        since: str | Unset = UNSET
        if not isinstance(self.since, Unset):
            since = self.since.isoformat()

        status = self.status

        tier = self.tier


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "created": created,
            "customer": customer,
        })
        if amount_cents is not UNSET:
            field_dict["amountCents"] = amount_cents
        if currency is not UNSET:
            field_dict["currency"] = currency
        if period is not UNSET:
            field_dict["period"] = period
        if since is not UNSET:
            field_dict["since"] = since
        if status is not UNSET:
            field_dict["status"] = status
        if tier is not UNSET:
            field_dict["tier"] = tier

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        created = datetime.datetime.fromisoformat(d.pop("created"))




        customer = d.pop("customer")

        amount_cents = d.pop("amountCents", UNSET)

        currency = d.pop("currency", UNSET)

        period = d.pop("period", UNSET)

        _since = d.pop("since", UNSET)
        since: datetime.datetime | Unset
        if isinstance(_since,  Unset):
            since = UNSET
        else:
            since = datetime.datetime.fromisoformat(_since)




        status = d.pop("status", UNSET)

        tier = d.pop("tier", UNSET)

        earn_import_candidate = cls(
            created=created,
            customer=customer,
            amount_cents=amount_cents,
            currency=currency,
            period=period,
            since=since,
            status=status,
            tier=tier,
        )

        return earn_import_candidate

