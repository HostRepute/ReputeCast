from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.revenue_credit import RevenueCredit





T = TypeVar("T", bound="RevenueTrials")



@_attrs_define
class RevenueTrials:
    """ 
        Attributes:
            became_paid (int):
            canceled (int):
            ended (int):
            ending_soon (int):
            started (int):
            trialing (int):
            ending_from (RevenueCredit | Unset):
     """

    became_paid: int
    canceled: int
    ended: int
    ending_soon: int
    started: int
    trialing: int
    ending_from: RevenueCredit | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.revenue_credit import RevenueCredit # noqa: PLC0415
        became_paid = self.became_paid

        canceled = self.canceled

        ended = self.ended

        ending_soon = self.ending_soon

        started = self.started

        trialing = self.trialing

        ending_from: dict[str, Any] | Unset = UNSET
        if not isinstance(self.ending_from, Unset):
            ending_from = self.ending_from.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "becamePaid": became_paid,
            "canceled": canceled,
            "ended": ended,
            "endingSoon": ending_soon,
            "started": started,
            "trialing": trialing,
        })
        if ending_from is not UNSET:
            field_dict["endingFrom"] = ending_from

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.revenue_credit import RevenueCredit # noqa: PLC0415
        d = dict(src_dict)
        became_paid = d.pop("becamePaid")

        canceled = d.pop("canceled")

        ended = d.pop("ended")

        ending_soon = d.pop("endingSoon")

        started = d.pop("started")

        trialing = d.pop("trialing")

        _ending_from = d.pop("endingFrom", UNSET)
        ending_from: RevenueCredit | Unset
        if isinstance(_ending_from,  Unset):
            ending_from = UNSET
        else:
            ending_from = RevenueCredit.from_dict(_ending_from)




        revenue_trials = cls(
            became_paid=became_paid,
            canceled=canceled,
            ended=ended,
            ending_soon=ending_soon,
            started=started,
            trialing=trialing,
            ending_from=ending_from,
        )

        return revenue_trials

