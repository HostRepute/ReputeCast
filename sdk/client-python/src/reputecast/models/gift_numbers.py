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
  from ..models.gift_months_count import GiftMonthsCount





T = TypeVar("T", bound="GiftNumbers")



@_attrs_define
class GiftNumbers:
    """ 
        Attributes:
            by_months (list[GiftMonthsCount]):
            delivered (int):
            redeemed (int):
            scheduled (int):
            sold (int):
            sold_cents (int):
            waiting (int):
            with_access (int):
            next_at (datetime.datetime | Unset):
            next_timezone (str | Unset):
     """

    by_months: list[GiftMonthsCount]
    delivered: int
    redeemed: int
    scheduled: int
    sold: int
    sold_cents: int
    waiting: int
    with_access: int
    next_at: datetime.datetime | Unset = UNSET
    next_timezone: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.gift_months_count import GiftMonthsCount # noqa: PLC0415
        by_months = []
        for by_months_item_data in self.by_months:
            by_months_item = by_months_item_data.to_dict()
            by_months.append(by_months_item)



        delivered = self.delivered

        redeemed = self.redeemed

        scheduled = self.scheduled

        sold = self.sold

        sold_cents = self.sold_cents

        waiting = self.waiting

        with_access = self.with_access

        next_at: str | Unset = UNSET
        if not isinstance(self.next_at, Unset):
            next_at = self.next_at.isoformat()

        next_timezone = self.next_timezone


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "byMonths": by_months,
            "delivered": delivered,
            "redeemed": redeemed,
            "scheduled": scheduled,
            "sold": sold,
            "soldCents": sold_cents,
            "waiting": waiting,
            "withAccess": with_access,
        })
        if next_at is not UNSET:
            field_dict["nextAt"] = next_at
        if next_timezone is not UNSET:
            field_dict["nextTimezone"] = next_timezone

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.gift_months_count import GiftMonthsCount # noqa: PLC0415
        d = dict(src_dict)
        by_months = []
        _by_months = d.pop("byMonths")
        for by_months_item_data in (_by_months):
            by_months_item = GiftMonthsCount.from_dict(by_months_item_data)



            by_months.append(by_months_item)


        delivered = d.pop("delivered")

        redeemed = d.pop("redeemed")

        scheduled = d.pop("scheduled")

        sold = d.pop("sold")

        sold_cents = d.pop("soldCents")

        waiting = d.pop("waiting")

        with_access = d.pop("withAccess")

        _next_at = d.pop("nextAt", UNSET)
        next_at: datetime.datetime | Unset
        if isinstance(_next_at,  Unset):
            next_at = UNSET
        else:
            next_at = datetime.datetime.fromisoformat(_next_at)




        next_timezone = d.pop("nextTimezone", UNSET)

        gift_numbers = cls(
            by_months=by_months,
            delivered=delivered,
            redeemed=redeemed,
            scheduled=scheduled,
            sold=sold,
            sold_cents=sold_cents,
            waiting=waiting,
            with_access=with_access,
            next_at=next_at,
            next_timezone=next_timezone,
        )

        return gift_numbers

