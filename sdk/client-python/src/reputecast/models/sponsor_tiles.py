from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="SponsorTiles")



@_attrs_define
class SponsorTiles:
    """ 
        Attributes:
            avg_click_issues (list[int]):
            booked_ahead_cents (int):
            booked_ahead_issues (list[int]):
            hold_ahead_cents (int):
            income_30_d_cents (int):
            income_issues (int):
            next_booked (int):
            next_count (int):
            next_from (int):
            next_hold (int):
            next_open (int):
            next_to (int):
            paid_30_d_cents (int):
            unpaid_30_d_cents (int):
            avg_click_rate (float | Unset):
     """

    avg_click_issues: list[int]
    booked_ahead_cents: int
    booked_ahead_issues: list[int]
    hold_ahead_cents: int
    income_30_d_cents: int
    income_issues: int
    next_booked: int
    next_count: int
    next_from: int
    next_hold: int
    next_open: int
    next_to: int
    paid_30_d_cents: int
    unpaid_30_d_cents: int
    avg_click_rate: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        avg_click_issues = self.avg_click_issues



        booked_ahead_cents = self.booked_ahead_cents

        booked_ahead_issues = self.booked_ahead_issues



        hold_ahead_cents = self.hold_ahead_cents

        income_30_d_cents = self.income_30_d_cents

        income_issues = self.income_issues

        next_booked = self.next_booked

        next_count = self.next_count

        next_from = self.next_from

        next_hold = self.next_hold

        next_open = self.next_open

        next_to = self.next_to

        paid_30_d_cents = self.paid_30_d_cents

        unpaid_30_d_cents = self.unpaid_30_d_cents

        avg_click_rate = self.avg_click_rate


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "avgClickIssues": avg_click_issues,
            "bookedAheadCents": booked_ahead_cents,
            "bookedAheadIssues": booked_ahead_issues,
            "holdAheadCents": hold_ahead_cents,
            "income30dCents": income_30_d_cents,
            "incomeIssues": income_issues,
            "nextBooked": next_booked,
            "nextCount": next_count,
            "nextFrom": next_from,
            "nextHold": next_hold,
            "nextOpen": next_open,
            "nextTo": next_to,
            "paid30dCents": paid_30_d_cents,
            "unpaid30dCents": unpaid_30_d_cents,
        })
        if avg_click_rate is not UNSET:
            field_dict["avgClickRate"] = avg_click_rate

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        avg_click_issues = cast(list[int], d.pop("avgClickIssues"))


        booked_ahead_cents = d.pop("bookedAheadCents")

        booked_ahead_issues = cast(list[int], d.pop("bookedAheadIssues"))


        hold_ahead_cents = d.pop("holdAheadCents")

        income_30_d_cents = d.pop("income30dCents")

        income_issues = d.pop("incomeIssues")

        next_booked = d.pop("nextBooked")

        next_count = d.pop("nextCount")

        next_from = d.pop("nextFrom")

        next_hold = d.pop("nextHold")

        next_open = d.pop("nextOpen")

        next_to = d.pop("nextTo")

        paid_30_d_cents = d.pop("paid30dCents")

        unpaid_30_d_cents = d.pop("unpaid30dCents")

        avg_click_rate = d.pop("avgClickRate", UNSET)

        sponsor_tiles = cls(
            avg_click_issues=avg_click_issues,
            booked_ahead_cents=booked_ahead_cents,
            booked_ahead_issues=booked_ahead_issues,
            hold_ahead_cents=hold_ahead_cents,
            income_30_d_cents=income_30_d_cents,
            income_issues=income_issues,
            next_booked=next_booked,
            next_count=next_count,
            next_from=next_from,
            next_hold=next_hold,
            next_open=next_open,
            next_to=next_to,
            paid_30_d_cents=paid_30_d_cents,
            unpaid_30_d_cents=unpaid_30_d_cents,
            avg_click_rate=avg_click_rate,
        )

        return sponsor_tiles

