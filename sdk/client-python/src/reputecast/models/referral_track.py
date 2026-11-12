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
  from ..models.referral_track_point import ReferralTrackPoint





T = TypeVar("T", bound="ReferralTrack")



@_attrs_define
class ReferralTrack:
    """ 
        Attributes:
            awaiting (int):
            counted (int):
            held (int):
            in_hold (int):
            next_by_seq (int):
            next_count (int):
            next_name (str):
            points (list[ReferralTrackPoint]):
            to_next (int):
            next_hold (datetime.datetime | Unset): When the next referral in hold counts
     """

    awaiting: int
    counted: int
    held: int
    in_hold: int
    next_by_seq: int
    next_count: int
    next_name: str
    points: list[ReferralTrackPoint]
    to_next: int
    next_hold: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.referral_track_point import ReferralTrackPoint # noqa: PLC0415
        awaiting = self.awaiting

        counted = self.counted

        held = self.held

        in_hold = self.in_hold

        next_by_seq = self.next_by_seq

        next_count = self.next_count

        next_name = self.next_name

        points = []
        for points_item_data in self.points:
            points_item = points_item_data.to_dict()
            points.append(points_item)



        to_next = self.to_next

        next_hold: str | Unset = UNSET
        if not isinstance(self.next_hold, Unset):
            next_hold = self.next_hold.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "awaiting": awaiting,
            "counted": counted,
            "held": held,
            "inHold": in_hold,
            "nextBySeq": next_by_seq,
            "nextCount": next_count,
            "nextName": next_name,
            "points": points,
            "toNext": to_next,
        })
        if next_hold is not UNSET:
            field_dict["nextHold"] = next_hold

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.referral_track_point import ReferralTrackPoint # noqa: PLC0415
        d = dict(src_dict)
        awaiting = d.pop("awaiting")

        counted = d.pop("counted")

        held = d.pop("held")

        in_hold = d.pop("inHold")

        next_by_seq = d.pop("nextBySeq")

        next_count = d.pop("nextCount")

        next_name = d.pop("nextName")

        points = []
        _points = d.pop("points")
        for points_item_data in (_points):
            points_item = ReferralTrackPoint.from_dict(points_item_data)



            points.append(points_item)


        to_next = d.pop("toNext")

        _next_hold = d.pop("nextHold", UNSET)
        next_hold: datetime.datetime | Unset
        if isinstance(_next_hold,  Unset):
            next_hold = UNSET
        else:
            next_hold = datetime.datetime.fromisoformat(_next_hold)




        referral_track = cls(
            awaiting=awaiting,
            counted=counted,
            held=held,
            in_hold=in_hold,
            next_by_seq=next_by_seq,
            next_count=next_count,
            next_name=next_name,
            points=points,
            to_next=to_next,
            next_hold=next_hold,
        )

        return referral_track

