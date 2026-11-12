from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.report_engagement_point import ReportEngagementPoint
  from ..models.report_rollout import ReportRollout





T = TypeVar("T", bound="ReportEngagement")



@_attrs_define
class ReportEngagement:
    """ 
        Attributes:
            available (bool):
            delivered (int):
            hours (int):
            points (list[ReportEngagementPoint]):
            human_within_6_h (float | Unset):
            rollout (ReportRollout | Unset):
     """

    available: bool
    delivered: int
    hours: int
    points: list[ReportEngagementPoint]
    human_within_6_h: float | Unset = UNSET
    rollout: ReportRollout | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.report_engagement_point import ReportEngagementPoint # noqa: PLC0415
        from ..models.report_rollout import ReportRollout # noqa: PLC0415
        available = self.available

        delivered = self.delivered

        hours = self.hours

        points = []
        for points_item_data in self.points:
            points_item = points_item_data.to_dict()
            points.append(points_item)



        human_within_6_h = self.human_within_6_h

        rollout: dict[str, Any] | Unset = UNSET
        if not isinstance(self.rollout, Unset):
            rollout = self.rollout.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "available": available,
            "delivered": delivered,
            "hours": hours,
            "points": points,
        })
        if human_within_6_h is not UNSET:
            field_dict["humanWithin6h"] = human_within_6_h
        if rollout is not UNSET:
            field_dict["rollout"] = rollout

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.report_engagement_point import ReportEngagementPoint # noqa: PLC0415
        from ..models.report_rollout import ReportRollout # noqa: PLC0415
        d = dict(src_dict)
        available = d.pop("available")

        delivered = d.pop("delivered")

        hours = d.pop("hours")

        points = []
        _points = d.pop("points")
        for points_item_data in (_points):
            points_item = ReportEngagementPoint.from_dict(points_item_data)



            points.append(points_item)


        human_within_6_h = d.pop("humanWithin6h", UNSET)

        _rollout = d.pop("rollout", UNSET)
        rollout: ReportRollout | Unset
        if isinstance(_rollout,  Unset):
            rollout = UNSET
        else:
            rollout = ReportRollout.from_dict(_rollout)




        report_engagement = cls(
            available=available,
            delivered=delivered,
            hours=hours,
            points=points,
            human_within_6_h=human_within_6_h,
            rollout=rollout,
        )

        return report_engagement

