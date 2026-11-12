from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.readiness_route_policy import ReadinessRoutePolicy
from ..models.readiness_route_stream import ReadinessRouteStream
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.readiness_held_move import ReadinessHeldMove
  from ..models.readiness_route_share import ReadinessRouteShare
  from ..models.readiness_timeline import ReadinessTimeline





T = TypeVar("T", bound="ReadinessRoute")



@_attrs_define
class ReadinessRoute:
    """ 
        Attributes:
            down (int):
            override (bool):
            policy (ReadinessRoutePolicy):
            shares (list[ReadinessRouteShare]):
            split_label (str):
            stream (ReadinessRouteStream):
            stream_label (str):
            total_cost (float):
            total_minutes (int):
            move (ReadinessHeldMove | Unset):
            timeline (ReadinessTimeline | Unset):
     """

    down: int
    override: bool
    policy: ReadinessRoutePolicy
    shares: list[ReadinessRouteShare]
    split_label: str
    stream: ReadinessRouteStream
    stream_label: str
    total_cost: float
    total_minutes: int
    move: ReadinessHeldMove | Unset = UNSET
    timeline: ReadinessTimeline | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.readiness_held_move import ReadinessHeldMove # noqa: PLC0415
        from ..models.readiness_route_share import ReadinessRouteShare # noqa: PLC0415
        from ..models.readiness_timeline import ReadinessTimeline # noqa: PLC0415
        down = self.down

        override = self.override

        policy = self.policy.value

        shares = []
        for shares_item_data in self.shares:
            shares_item = shares_item_data.to_dict()
            shares.append(shares_item)



        split_label = self.split_label

        stream = self.stream.value

        stream_label = self.stream_label

        total_cost = self.total_cost

        total_minutes = self.total_minutes

        move: dict[str, Any] | Unset = UNSET
        if not isinstance(self.move, Unset):
            move = self.move.to_dict()

        timeline: dict[str, Any] | Unset = UNSET
        if not isinstance(self.timeline, Unset):
            timeline = self.timeline.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "down": down,
            "override": override,
            "policy": policy,
            "shares": shares,
            "splitLabel": split_label,
            "stream": stream,
            "streamLabel": stream_label,
            "totalCost": total_cost,
            "totalMinutes": total_minutes,
        })
        if move is not UNSET:
            field_dict["move"] = move
        if timeline is not UNSET:
            field_dict["timeline"] = timeline

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.readiness_held_move import ReadinessHeldMove # noqa: PLC0415
        from ..models.readiness_route_share import ReadinessRouteShare # noqa: PLC0415
        from ..models.readiness_timeline import ReadinessTimeline # noqa: PLC0415
        d = dict(src_dict)
        down = d.pop("down")

        override = d.pop("override")

        policy = ReadinessRoutePolicy(d.pop("policy"))




        shares = []
        _shares = d.pop("shares")
        for shares_item_data in (_shares):
            shares_item = ReadinessRouteShare.from_dict(shares_item_data)



            shares.append(shares_item)


        split_label = d.pop("splitLabel")

        stream = ReadinessRouteStream(d.pop("stream"))




        stream_label = d.pop("streamLabel")

        total_cost = d.pop("totalCost")

        total_minutes = d.pop("totalMinutes")

        _move = d.pop("move", UNSET)
        move: ReadinessHeldMove | Unset
        if isinstance(_move,  Unset):
            move = UNSET
        else:
            move = ReadinessHeldMove.from_dict(_move)




        _timeline = d.pop("timeline", UNSET)
        timeline: ReadinessTimeline | Unset
        if isinstance(_timeline,  Unset):
            timeline = UNSET
        else:
            timeline = ReadinessTimeline.from_dict(_timeline)




        readiness_route = cls(
            down=down,
            override=override,
            policy=policy,
            shares=shares,
            split_label=split_label,
            stream=stream,
            stream_label=stream_label,
            total_cost=total_cost,
            total_minutes=total_minutes,
            move=move,
            timeline=timeline,
        )

        return readiness_route

