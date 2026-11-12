from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="CommerceTriggerStats")



@_attrs_define
class CommerceTriggerStats:
    """ 
        Attributes:
            abandoned (int):
            currency (str):
            entered (int):
            no_consent (int):
            orders (int):
            revenue (int):
            skipped (int):
            with_email (int):
            since (datetime.datetime | Unset):
     """

    abandoned: int
    currency: str
    entered: int
    no_consent: int
    orders: int
    revenue: int
    skipped: int
    with_email: int
    since: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        abandoned = self.abandoned

        currency = self.currency

        entered = self.entered

        no_consent = self.no_consent

        orders = self.orders

        revenue = self.revenue

        skipped = self.skipped

        with_email = self.with_email

        since: str | Unset = UNSET
        if not isinstance(self.since, Unset):
            since = self.since.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "abandoned": abandoned,
            "currency": currency,
            "entered": entered,
            "noConsent": no_consent,
            "orders": orders,
            "revenue": revenue,
            "skipped": skipped,
            "withEmail": with_email,
        })
        if since is not UNSET:
            field_dict["since"] = since

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        abandoned = d.pop("abandoned")

        currency = d.pop("currency")

        entered = d.pop("entered")

        no_consent = d.pop("noConsent")

        orders = d.pop("orders")

        revenue = d.pop("revenue")

        skipped = d.pop("skipped")

        with_email = d.pop("withEmail")

        _since = d.pop("since", UNSET)
        since: datetime.datetime | Unset
        if isinstance(_since,  Unset):
            since = UNSET
        else:
            since = datetime.datetime.fromisoformat(_since)




        commerce_trigger_stats = cls(
            abandoned=abandoned,
            currency=currency,
            entered=entered,
            no_consent=no_consent,
            orders=orders,
            revenue=revenue,
            skipped=skipped,
            with_email=with_email,
            since=since,
        )

        return commerce_trigger_stats

