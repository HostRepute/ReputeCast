from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.revenue_conversion_line import RevenueConversionLine
  from ..models.revenue_conversion_row import RevenueConversionRow
  from ..models.revenue_move import RevenueMove





T = TypeVar("T", bound="RevenueConversions")



@_attrs_define
class RevenueConversions:
    """ 
        Attributes:
            by_email (list[RevenueConversionRow]):
            count (int):
            currency (str):
            latest (list[RevenueConversionLine]):
            other_currency (int):
            total (int):
            uncredited (RevenueMove):
            value_cents (int):
            name (str | Unset):
            sender (str | Unset):
            since (datetime.datetime | Unset):
            typical_cents (int | Unset):
     """

    by_email: list[RevenueConversionRow]
    count: int
    currency: str
    latest: list[RevenueConversionLine]
    other_currency: int
    total: int
    uncredited: RevenueMove
    value_cents: int
    name: str | Unset = UNSET
    sender: str | Unset = UNSET
    since: datetime.datetime | Unset = UNSET
    typical_cents: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.revenue_conversion_line import RevenueConversionLine # noqa: PLC0415
        from ..models.revenue_conversion_row import RevenueConversionRow # noqa: PLC0415
        from ..models.revenue_move import RevenueMove # noqa: PLC0415
        by_email = []
        for by_email_item_data in self.by_email:
            by_email_item = by_email_item_data.to_dict()
            by_email.append(by_email_item)



        count = self.count

        currency = self.currency

        latest = []
        for latest_item_data in self.latest:
            latest_item = latest_item_data.to_dict()
            latest.append(latest_item)



        other_currency = self.other_currency

        total = self.total

        uncredited = self.uncredited.to_dict()

        value_cents = self.value_cents

        name = self.name

        sender = self.sender

        since: str | Unset = UNSET
        if not isinstance(self.since, Unset):
            since = self.since.isoformat()

        typical_cents = self.typical_cents


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "byEmail": by_email,
            "count": count,
            "currency": currency,
            "latest": latest,
            "otherCurrency": other_currency,
            "total": total,
            "uncredited": uncredited,
            "valueCents": value_cents,
        })
        if name is not UNSET:
            field_dict["name"] = name
        if sender is not UNSET:
            field_dict["sender"] = sender
        if since is not UNSET:
            field_dict["since"] = since
        if typical_cents is not UNSET:
            field_dict["typicalCents"] = typical_cents

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.revenue_conversion_line import RevenueConversionLine # noqa: PLC0415
        from ..models.revenue_conversion_row import RevenueConversionRow # noqa: PLC0415
        from ..models.revenue_move import RevenueMove # noqa: PLC0415
        d = dict(src_dict)
        by_email = []
        _by_email = d.pop("byEmail")
        for by_email_item_data in (_by_email):
            by_email_item = RevenueConversionRow.from_dict(by_email_item_data)



            by_email.append(by_email_item)


        count = d.pop("count")

        currency = d.pop("currency")

        latest = []
        _latest = d.pop("latest")
        for latest_item_data in (_latest):
            latest_item = RevenueConversionLine.from_dict(latest_item_data)



            latest.append(latest_item)


        other_currency = d.pop("otherCurrency")

        total = d.pop("total")

        uncredited = RevenueMove.from_dict(d.pop("uncredited"))




        value_cents = d.pop("valueCents")

        name = d.pop("name", UNSET)

        sender = d.pop("sender", UNSET)

        _since = d.pop("since", UNSET)
        since: datetime.datetime | Unset
        if isinstance(_since,  Unset):
            since = UNSET
        else:
            since = datetime.datetime.fromisoformat(_since)




        typical_cents = d.pop("typicalCents", UNSET)

        revenue_conversions = cls(
            by_email=by_email,
            count=count,
            currency=currency,
            latest=latest,
            other_currency=other_currency,
            total=total,
            uncredited=uncredited,
            value_cents=value_cents,
            name=name,
            sender=sender,
            since=since,
            typical_cents=typical_cents,
        )

        return revenue_conversions

