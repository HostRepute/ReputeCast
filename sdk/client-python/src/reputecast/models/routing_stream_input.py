from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.routing_stream_input_failover import RoutingStreamInputFailover
from ..models.routing_stream_input_stream import RoutingStreamInputStream
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.routing_condition import RoutingCondition
  from ..models.routing_split import RoutingSplit





T = TypeVar("T", bound="RoutingStreamInput")



@_attrs_define
class RoutingStreamInput:
    """ 
        Attributes:
            conditions (list[RoutingCondition]):
            failover (RoutingStreamInputFailover):
            splits (list[RoutingSplit]):
            stream (RoutingStreamInputStream):
            base_version (str | Unset):
            fallback (str | Unset): Provider id, or empty for none
            priority (bool | Unset):
     """

    conditions: list[RoutingCondition]
    failover: RoutingStreamInputFailover
    splits: list[RoutingSplit]
    stream: RoutingStreamInputStream
    base_version: str | Unset = UNSET
    fallback: str | Unset = UNSET
    priority: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.routing_condition import RoutingCondition # noqa: PLC0415
        from ..models.routing_split import RoutingSplit # noqa: PLC0415
        conditions = []
        for conditions_item_data in self.conditions:
            conditions_item = conditions_item_data.to_dict()
            conditions.append(conditions_item)



        failover = self.failover.value

        splits = []
        for splits_item_data in self.splits:
            splits_item = splits_item_data.to_dict()
            splits.append(splits_item)



        stream = self.stream.value

        base_version = self.base_version

        fallback = self.fallback

        priority = self.priority


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "conditions": conditions,
            "failover": failover,
            "splits": splits,
            "stream": stream,
        })
        if base_version is not UNSET:
            field_dict["baseVersion"] = base_version
        if fallback is not UNSET:
            field_dict["fallback"] = fallback
        if priority is not UNSET:
            field_dict["priority"] = priority

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.routing_condition import RoutingCondition # noqa: PLC0415
        from ..models.routing_split import RoutingSplit # noqa: PLC0415
        d = dict(src_dict)
        conditions = []
        _conditions = d.pop("conditions")
        for conditions_item_data in (_conditions):
            conditions_item = RoutingCondition.from_dict(conditions_item_data)



            conditions.append(conditions_item)


        failover = RoutingStreamInputFailover(d.pop("failover"))




        splits = []
        _splits = d.pop("splits")
        for splits_item_data in (_splits):
            splits_item = RoutingSplit.from_dict(splits_item_data)



            splits.append(splits_item)


        stream = RoutingStreamInputStream(d.pop("stream"))




        base_version = d.pop("baseVersion", UNSET)

        fallback = d.pop("fallback", UNSET)

        priority = d.pop("priority", UNSET)

        routing_stream_input = cls(
            conditions=conditions,
            failover=failover,
            splits=splits,
            stream=stream,
            base_version=base_version,
            fallback=fallback,
            priority=priority,
        )

        return routing_stream_input

