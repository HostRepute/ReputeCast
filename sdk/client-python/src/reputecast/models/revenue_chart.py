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
  from ..models.revenue_mark import RevenueMark
  from ..models.revenue_point import RevenuePoint





T = TypeVar("T", bound="RevenueChart")



@_attrs_define
class RevenueChart:
    """ 
        Attributes:
            delta_cents (int):
            end_cents (int):
            marks (list[RevenueMark]):
            points (list[RevenuePoint]):
            since (datetime.datetime):
            start_cents (int):
            delta_pct (float | Unset):
            history_until (datetime.datetime | Unset):
     """

    delta_cents: int
    end_cents: int
    marks: list[RevenueMark]
    points: list[RevenuePoint]
    since: datetime.datetime
    start_cents: int
    delta_pct: float | Unset = UNSET
    history_until: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.revenue_mark import RevenueMark # noqa: PLC0415
        from ..models.revenue_point import RevenuePoint # noqa: PLC0415
        delta_cents = self.delta_cents

        end_cents = self.end_cents

        marks = []
        for marks_item_data in self.marks:
            marks_item = marks_item_data.to_dict()
            marks.append(marks_item)



        points = []
        for points_item_data in self.points:
            points_item = points_item_data.to_dict()
            points.append(points_item)



        since = self.since.isoformat()

        start_cents = self.start_cents

        delta_pct = self.delta_pct

        history_until: str | Unset = UNSET
        if not isinstance(self.history_until, Unset):
            history_until = self.history_until.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "deltaCents": delta_cents,
            "endCents": end_cents,
            "marks": marks,
            "points": points,
            "since": since,
            "startCents": start_cents,
        })
        if delta_pct is not UNSET:
            field_dict["deltaPct"] = delta_pct
        if history_until is not UNSET:
            field_dict["historyUntil"] = history_until

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.revenue_mark import RevenueMark # noqa: PLC0415
        from ..models.revenue_point import RevenuePoint # noqa: PLC0415
        d = dict(src_dict)
        delta_cents = d.pop("deltaCents")

        end_cents = d.pop("endCents")

        marks = []
        _marks = d.pop("marks")
        for marks_item_data in (_marks):
            marks_item = RevenueMark.from_dict(marks_item_data)



            marks.append(marks_item)


        points = []
        _points = d.pop("points")
        for points_item_data in (_points):
            points_item = RevenuePoint.from_dict(points_item_data)



            points.append(points_item)


        since = datetime.datetime.fromisoformat(d.pop("since"))




        start_cents = d.pop("startCents")

        delta_pct = d.pop("deltaPct", UNSET)

        _history_until = d.pop("historyUntil", UNSET)
        history_until: datetime.datetime | Unset
        if isinstance(_history_until,  Unset):
            history_until = UNSET
        else:
            history_until = datetime.datetime.fromisoformat(_history_until)




        revenue_chart = cls(
            delta_cents=delta_cents,
            end_cents=end_cents,
            marks=marks,
            points=points,
            since=since,
            start_cents=start_cents,
            delta_pct=delta_pct,
            history_until=history_until,
        )

        return revenue_chart

